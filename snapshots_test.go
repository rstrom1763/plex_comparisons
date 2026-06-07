package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func TestReadOrCreateLocalMovieSnapshotFallsBackToStaleSnapshotWhenRefreshFails(t *testing.T) {
	chdirTemp(t)

	savedMovies := []*structs.Movie{{Title: "Saved Local", Year: 2001}}
	writeTestMovieSnapshot(t, localSnapshotID, savedMovies)
	markSnapshotOld(t, localSnapshotID)

	db := newServerFixtureDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("Close(db) error = %v", err)
	}

	got, err := readOrCreateLocalMovieSnapshot(db)
	if err != nil {
		t.Fatalf("readOrCreateLocalMovieSnapshot() error = %v", err)
	}
	if len(got) != 1 || got[0].Title != "Saved Local" {
		t.Fatalf("readOrCreateLocalMovieSnapshot() = %+v, want stale saved movie", got)
	}
}

func TestReadOrCreateRemoteMovieSnapshotRefreshesStaleSnapshot(t *testing.T) {
	chdirTemp(t)

	const snapshotID = "server_1"
	writeTestMovieSnapshot(t, snapshotID, []*structs.Movie{{Title: "Saved Remote", Year: 2001}})
	markSnapshotOld(t, snapshotID)
	updatedMovies := []*structs.Movie{{Title: "Updated Remote", Year: 2026}}
	compressedSnapshot := gzipTestMovies(t, updatedMovies)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Encoding"); got != "gzip" {
			t.Fatalf("Accept-Encoding = %q, want gzip", got)
		}
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressedSnapshot)
	}))
	t.Cleanup(remote.Close)

	got, err := readOrCreateRemoteMovieSnapshot(snapshotSource{ID: snapshotID, Address: remote.URL})
	if err != nil {
		t.Fatalf("readOrCreateRemoteMovieSnapshot() error = %v", err)
	}
	if len(got) != 1 || got[0].Title != "Updated Remote" {
		t.Fatalf("readOrCreateRemoteMovieSnapshot() = %+v, want updated movie", got)
	}

	cachedSnapshot, err := os.ReadFile(movieSnapshotPath(snapshotID))
	if err != nil {
		t.Fatalf("ReadFile(snapshot) error = %v", err)
	}
	if !bytes.Equal(cachedSnapshot, compressedSnapshot) {
		t.Fatal("cached snapshot bytes do not match compressed remote response")
	}
}

func TestReadOrCreateRemoteMovieSnapshotFallsBackToStaleSnapshotWhenRefreshFails(t *testing.T) {
	chdirTemp(t)

	const snapshotID = "server_1"
	writeTestMovieSnapshot(t, snapshotID, []*structs.Movie{{Title: "Saved Remote", Year: 2001}})
	markSnapshotOld(t, snapshotID)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(remote.Close)

	got, err := readOrCreateRemoteMovieSnapshot(snapshotSource{ID: snapshotID, Address: remote.URL})
	if err != nil {
		t.Fatalf("readOrCreateRemoteMovieSnapshot() error = %v", err)
	}
	if len(got) != 1 || got[0].Title != "Saved Remote" {
		t.Fatalf("readOrCreateRemoteMovieSnapshot() = %+v, want stale saved movie", got)
	}
}

func writeTestMovieSnapshot(t *testing.T, serverID string, movies []*structs.Movie) {
	t.Helper()

	data, err := json.Marshal(movies)
	if err != nil {
		t.Fatalf("Marshal(movies) error = %v", err)
	}
	if err := writeMovieSnapshot(serverID, data); err != nil {
		t.Fatalf("writeMovieSnapshot() error = %v", err)
	}
}

func gzipTestMovies(t *testing.T, movies []*structs.Movie) []byte {
	t.Helper()

	data, err := json.Marshal(movies)
	if err != nil {
		t.Fatalf("Marshal(movies) error = %v", err)
	}

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("Write(gzip) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close(gzip) error = %v", err)
	}
	return buf.Bytes()
}

func markSnapshotOld(t *testing.T, serverID string) {
	t.Helper()

	oldTime := time.Now().Add(-(constants.SNAPSHOT_MAX_AGE + time.Hour))
	if err := os.Chtimes(movieSnapshotPath(serverID), oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(snapshot) error = %v", err)
	}
}
