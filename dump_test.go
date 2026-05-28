package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func newTestPlexDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	schema := []string{
		`CREATE TABLE metadata_items (
			id INTEGER PRIMARY KEY,
			title TEXT,
			content_rating TEXT,
			year INTEGER,
			tags_genre TEXT,
			rating REAL,
			audience_rating REAL,
			hash TEXT
		);`,
		`CREATE TABLE library_sections (
			id INTEGER PRIMARY KEY,
			name TEXT,
			section_type INTEGER
		);`,
		`CREATE TABLE media_items (
			id INTEGER PRIMARY KEY,
			metadata_item_id INTEGER,
			library_section_id INTEGER,
			container TEXT,
			bitrate INTEGER,
			video_codec TEXT,
			height INTEGER,
			width INTEGER,
			audio_codec TEXT
		);`,
		`CREATE TABLE media_parts (
			id INTEGER PRIMARY KEY,
			media_item_id INTEGER,
			file TEXT,
			hash TEXT,
			size INTEGER,
			duration INTEGER
		);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema Exec() error = %v", err)
		}
	}

	if _, err := db.Exec(`
		INSERT INTO metadata_items (
			id, title, content_rating, year, tags_genre, rating, audience_rating, hash
		) VALUES (
			1, 'Alien', 'R', 1979, 'Horror', 8.5, 9.1, 'metadata-hash'
		);
		INSERT INTO library_sections (id, name, section_type)
		VALUES (1, 'Movies', 1);
		INSERT INTO media_items (
			id, metadata_item_id, library_section_id, container, bitrate,
			video_codec, height, width, audio_codec
		) VALUES (
			1, 1, 1, 'mkv', 5000, 'h264', 1080, 1920, 'aac'
		);
		INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
		VALUES (1, 1, '/media/Alien.mkv', 'file-hash', 123456, 7000000);
	`); err != nil {
		t.Fatalf("fixture Exec() error = %v", err)
	}

	return db
}

func TestGetMoviesFromPlexDB(t *testing.T) {
	db := newTestPlexDB(t)

	movies, err := structs.GetMovies(db)
	if err != nil {
		t.Fatalf("GetMovies() error = %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}

	got := movies[0]
	if got.Title != "Alien" {
		t.Fatalf("Title = %q, want %q", got.Title, "Alien")
	}
	if got.MetadataHash != "metadata-hash" {
		t.Fatalf("MetadataHash = %q, want %q", got.MetadataHash, "metadata-hash")
	}
	if got.Resolution != "1080p" {
		t.Fatalf("Resolution = %q, want %q", got.Resolution, "1080p")
	}
}

func TestDumpReturnsInitDBError(t *testing.T) {
	err := dump(filepath.Join(t.TempDir(), "missing", "plex.db"))
	if err == nil {
		t.Fatal("dump() error = nil, want error")
	}
}

func TestDumpWritesCSVFiles(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "plex.db")
	createDumpFixtureDB(t, dbPath)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(temp dir) error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Chdir(original dir) error = %v", err)
		}
	})

	if err := dump(dbPath); err != nil {
		t.Fatalf("dump() error = %v", err)
	}

	for _, name := range []string{"movies.csv", "songs.csv", "episodes.csv"} {
		contents, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		if len(contents) == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
}

func createDumpFixtureDB(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	statements := []string{
		`CREATE TABLE metadata_items (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			metadata_type INTEGER,
			"index" INTEGER,
			title TEXT,
			content_rating TEXT,
			year INTEGER,
			tags_genre TEXT,
			rating REAL,
			audience_rating REAL,
			hash TEXT
		);`,
		`CREATE TABLE library_sections (
			id INTEGER PRIMARY KEY,
			name TEXT,
			section_type INTEGER
		);`,
		`CREATE TABLE media_items (
			id INTEGER PRIMARY KEY,
			metadata_item_id INTEGER,
			library_section_id INTEGER,
			container TEXT,
			bitrate INTEGER,
			video_codec TEXT,
			height INTEGER,
			width INTEGER,
			audio_codec TEXT
		);`,
		`CREATE TABLE media_parts (
			id INTEGER PRIMARY KEY,
			media_item_id INTEGER,
			file TEXT,
			hash TEXT,
			size INTEGER,
			duration INTEGER
		);`,
		`INSERT INTO library_sections (id, name, section_type)
			VALUES (1, 'Movies', 1), (2, 'TV', 2), (3, 'Music', 8);`,
		`INSERT INTO metadata_items (
			id, metadata_type, title, content_rating, year, tags_genre,
			rating, audience_rating, hash
		) VALUES (
			10, 1, 'Alien', 'R', 1979, 'Horror', 8.5, 9.1, 'movie-metadata'
		);`,
		`INSERT INTO media_items (
			id, metadata_item_id, library_section_id, container, bitrate,
			video_codec, height, width, audio_codec
		) VALUES (
			10, 10, 1, 'mkv', 5000, 'h264', 1080, 1920, 'aac'
		);`,
		`INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
			VALUES (10, 10, '/media/Alien.mkv', 'movie-file', 123456, 7000000);`,
		`INSERT INTO metadata_items (id, metadata_type, title, year, tags_genre, hash)
			VALUES (20, 8, 'Artist', 1990, 'Rock', 'artist-metadata');`,
		`INSERT INTO metadata_items (id, parent_id, metadata_type, title, year, tags_genre, hash)
			VALUES (21, 20, 9, 'Album', 1999, 'Rock', 'album-metadata');`,
		`INSERT INTO metadata_items (id, parent_id, metadata_type, title, year, tags_genre, hash)
			VALUES (22, 21, 10, 'Song', 2000, 'Rock', 'song-metadata');`,
		`INSERT INTO media_items (id, metadata_item_id, library_section_id, bitrate, audio_codec)
			VALUES (22, 22, 3, 900, 'flac');`,
		`INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
			VALUES (22, 22, '/media/song.flac', 'song-file', 1000, 200);`,
		`INSERT INTO metadata_items (id, metadata_type, title, year, rating, audience_rating, hash)
			VALUES (30, 2, 'Show', 2020, 8.0, 8.5, 'show-metadata');`,
		`INSERT INTO metadata_items (id, parent_id, metadata_type, "index", title, year, hash)
			VALUES (31, 30, 3, 1, 'Season 1', 2020, 'season-metadata');`,
		`INSERT INTO metadata_items (
			id, parent_id, metadata_type, "index", title, content_rating, year, hash
		) VALUES (
			32, 31, 4, 2, 'Episode', 'TV-14', 2020, 'episode-metadata'
		);`,
		`INSERT INTO media_items (
			id, metadata_item_id, library_section_id, container, bitrate,
			video_codec, height, width, audio_codec
		) VALUES (
			32, 32, 2, 'mkv', 4000, 'h264', 720, 1280, 'aac'
		);`,
		`INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
			VALUES (32, 32, '/media/show.mkv', 'episode-file', 2000, 3000);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("Exec(%q) error = %v", statement, err)
		}
	}
}
