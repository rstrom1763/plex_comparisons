package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/tls"

	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

const (
	cacheDir        = "cache"
	snapshotDir     = "snapshots"
	thumbnailDir    = "thumb_cache"
	localSnapshotID = "local"
)

type snapshotSource struct {
	ID      string
	Address string
	Token   string
}

type remoteServerError struct {
	status  int
	message string
}

func (e *remoteServerError) Error() string {
	return e.message
}

func ensureCacheDirs() error {
	if err := os.MkdirAll(snapshotCacheDir(), 0755); err != nil {
		return fmt.Errorf("could not create snapshot cache directory: %w", err)
	}
	if err := migrateLegacyThumbnailCache(); err != nil {
		return err
	}
	if err := os.MkdirAll(thumbnailCacheDir(), 0755); err != nil {
		return fmt.Errorf("could not create thumbnail cache directory: %w", err)
	}
	return nil
}

func migrateLegacyThumbnailCache() error {
	const legacyThumbCache = "thumb_cache"
	if _, err := os.Stat(legacyThumbCache); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("could not inspect legacy thumbnail cache: %w", err)
	}
	if _, err := os.Stat(thumbnailCacheDir()); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not inspect thumbnail cache directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("could not create cache directory: %w", err)
	}
	if err := os.Rename(legacyThumbCache, thumbnailCacheDir()); err != nil {
		return fmt.Errorf("could not move legacy thumbnail cache: %w", err)
	}
	return nil
}

func snapshotCacheDir() string {
	return filepath.Join(cacheDir, snapshotDir)
}

func thumbnailCacheDir() string {
	return filepath.Join(cacheDir, thumbnailDir)
}

func movieSnapshotPath(serverID string) string {
	return filepath.Join(snapshotCacheDir(), fmt.Sprintf("%s_movies.json.gz", serverID))
}

func readOrCreateLocalMovieSnapshot(db *sql.DB) ([]*structs.Movie, error) {
	movies, err := readMovieSnapshot(localSnapshotID)
	if err == nil {
		if snapshotIsStale(localSnapshotID) {
			if updatedMovies, err := refreshLocalMovieSnapshot(db); err == nil {
				return updatedMovies, nil
			}
		}
		return movies, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if err := takeLocalMovieSnapshot(db); err != nil {
		return nil, err
	}
	return readMovieSnapshot(localSnapshotID)
}

func readOrCreateRemoteMovieSnapshot(source snapshotSource) ([]*structs.Movie, error) {
	movies, err := readMovieSnapshot(source.ID)
	if err == nil {
		if snapshotIsStale(source.ID) {
			if updatedMovies, err := refreshRemoteMovieSnapshot(source); err == nil {
				return updatedMovies, nil
			}
		}
		return movies, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if err := takeRemoteMovieSnapshot(source); err != nil {
		return nil, err
	}
	return readMovieSnapshot(source.ID)
}

func takeLocalMovieSnapshot(db *sql.DB) error {
	_, err := refreshLocalMovieSnapshot(db)
	return err
}

func refreshLocalMovieSnapshot(db *sql.DB) ([]*structs.Movie, error) {
	movies, err := structs.GetMovies(db)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(movies)
	if err != nil {
		return nil, fmt.Errorf("could not marshal local movie snapshot: %w", err)
	}
	if err := writeMovieSnapshot(localSnapshotID, data); err != nil {
		return nil, err
	}
	return movies, nil
}

func takeRemoteMovieSnapshot(source snapshotSource) error {
	_, err := refreshRemoteMovieSnapshot(source)
	return err
}

func refreshRemoteMovieSnapshot(source snapshotSource) ([]*structs.Movie, error) {
	if err := os.MkdirAll(snapshotCacheDir(), 0755); err != nil {
		return nil, fmt.Errorf("could not create snapshot cache directory: %w", err)
	}

	tempFile, err := os.CreateTemp(snapshotCacheDir(), source.ID+"_*.json.gz")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary remote movie snapshot: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if err := fetchRemoteMovieSnapshotToFile(source, tempFile); err != nil {
		_ = tempFile.Close()
		return nil, err
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("could not close temporary remote movie snapshot: %w", err)
	}

	movies, err := readMovieSnapshotFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("could not decode remote movie snapshot: %w", err)
	}

	if err := os.Rename(tempPath, movieSnapshotPath(source.ID)); err != nil {
		return nil, fmt.Errorf("could not replace remote movie snapshot: %w", err)
	}

	return movies, nil
}

func fetchRemoteMovieSnapshotToFile(source snapshotSource, writer io.Writer) error {
	req, err := http.NewRequest("GET", strings.TrimRight(source.Address, "/")+"/api/movies", nil)
	if err != nil {
		return &remoteServerError{status: http.StatusInternalServerError, message: "could not create request: " + err.Error()}
	}
	req.Header.Set("Accept-Encoding", "gzip")
	if source.Token != "" {
		req.Header.Set("X-Server-Token", source.Token)
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}}
	resp, err := client.Do(req)
	if err != nil {
		return &remoteServerError{status: http.StatusBadGateway, message: "could not reach remote server: " + err.Error()}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		return &remoteServerError{status: http.StatusUnauthorized, message: "remote server rejected authentication. Ensure tokens match."}
	}
	if resp.StatusCode != http.StatusOK {
		return &remoteServerError{status: resp.StatusCode, message: "remote server returned error"}
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return &remoteServerError{status: http.StatusInternalServerError, message: "failed to read remote movies: " + err.Error()}
	}
	return nil
}

func writeMovieSnapshot(serverID string, jsonData []byte) error {
	if err := os.MkdirAll(snapshotCacheDir(), 0755); err != nil {
		return fmt.Errorf("could not create snapshot cache directory: %w", err)
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		_ = gzipWriter.Close()
		return fmt.Errorf("could not gzip movie snapshot: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("could not finish movie snapshot gzip: %w", err)
	}

	if err := os.WriteFile(movieSnapshotPath(serverID), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("could not write movie snapshot: %w", err)
	}
	return nil
}

func readMovieSnapshot(serverID string) ([]*structs.Movie, error) {
	return readMovieSnapshotFile(movieSnapshotPath(serverID))
}

func readMovieSnapshotFile(path string) ([]*structs.Movie, error) {
	data, err := readMovieSnapshotFileBytes(path)
	if err != nil {
		return nil, err
	}

	var movies []*structs.Movie
	if err := json.Unmarshal(data, &movies); err != nil {
		return nil, fmt.Errorf("could not decode movie snapshot: %w", err)
	}
	return movies, nil
}

func readMovieSnapshotBytes(serverID string) ([]byte, error) {
	return readMovieSnapshotFileBytes(movieSnapshotPath(serverID))
}

func readMovieSnapshotFileBytes(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("could not read gzipped movie snapshot: %w", err)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	return io.ReadAll(gzipReader)
}

func snapshotTakenAt(serverID string) *time.Time {
	info, err := os.Stat(movieSnapshotPath(serverID))
	if err != nil {
		return nil
	}
	modTime := info.ModTime()
	return &modTime
}

func snapshotAge(serverID string) string {
	takenAt := snapshotTakenAt(serverID)
	if takenAt == nil {
		return "never"
	}
	return humanDuration(time.Since(*takenAt)) + " ago"
}

func snapshotIsStale(serverID string) bool {
	takenAt := snapshotTakenAt(serverID)
	return takenAt != nil && time.Since(*takenAt) > constants.SNAPSHOT_MAX_AGE
}

func humanDuration(duration time.Duration) string {
	if duration < time.Minute {
		return "0 minutes"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
