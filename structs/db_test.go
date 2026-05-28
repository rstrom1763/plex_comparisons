package structs

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newMediaTestDB(t *testing.T) *sql.DB {
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

		`INSERT INTO metadata_items (
			id, metadata_type, title, year, tags_genre, hash
		) VALUES (
			20, 8, 'Artist', 1990, 'Rock', 'artist-metadata'
		);`,
		`INSERT INTO metadata_items (
			id, parent_id, metadata_type, title, year, tags_genre, hash
		) VALUES (
			21, 20, 9, 'Album', 1999, 'Rock', 'album-metadata'
		);`,
		`INSERT INTO metadata_items (
			id, parent_id, metadata_type, title, year, tags_genre, hash
		) VALUES (
			22, 21, 10, 'Song', 2000, 'Rock', 'song-metadata'
		);`,
		`INSERT INTO media_items (
			id, metadata_item_id, library_section_id, bitrate, audio_codec
		) VALUES (
			22, 22, 3, 900, 'flac'
		);`,
		`INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
			VALUES (22, 22, '/media/song.flac', 'song-file', 1000, 200);`,

		`INSERT INTO metadata_items (
			id, metadata_type, title, year, rating, audience_rating, hash
		) VALUES (
			30, 2, 'Show', 2020, 8.0, 8.5, 'show-metadata'
		);`,
		`INSERT INTO metadata_items (
			id, parent_id, metadata_type, "index", title, year, hash
		) VALUES (
			31, 30, 3, 1, 'Season 1', 2020, 'season-metadata'
		);`,
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

	return db
}

func TestGetMovies(t *testing.T) {
	movies, err := GetMovies(newMediaTestDB(t))
	if err != nil {
		t.Fatalf("GetMovies() error = %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}

	got := movies[0]
	if got.Title != "Alien" || got.MetadataHash != "movie-metadata" || got.Resolution != "1080p" {
		t.Fatalf("movie = %+v, want Alien fixture", got)
	}
	if got.QualityScore == 0 {
		t.Fatal("QualityScore = 0, want calculated score")
	}
}

func TestGetSongs(t *testing.T) {
	songs, err := GetSongs(newMediaTestDB(t))
	if err != nil {
		t.Fatalf("GetSongs() error = %v", err)
	}
	if len(songs) != 1 {
		t.Fatalf("len(songs) = %d, want 1", len(songs))
	}

	got := songs[0]
	if got.Title != "Song" || got.AlbumTitle != "Album" || got.ArtistName != "Artist" {
		t.Fatalf("song = %+v, want Song fixture", got)
	}
}

func TestGetEpisodes(t *testing.T) {
	episodes, err := GetEpisodes(newMediaTestDB(t))
	if err != nil {
		t.Fatalf("GetEpisodes() error = %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("len(episodes) = %d, want 1", len(episodes))
	}

	got := episodes[0]
	if got.ShowTitle != "Show" || got.SeasonNumber != 1 || got.EpisodeNumber != 2 || got.EpisodeTitle != "Episode" {
		t.Fatalf("episode = %+v, want Episode fixture", got)
	}
	if got.MetadataHash != "episode-metadata" {
		t.Fatalf("MetadataHash = %q, want %q", got.MetadataHash, "episode-metadata")
	}
}

func TestMediaDBQueryErrors(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := GetMovies(db); err == nil {
		t.Fatal("GetMovies() error = nil, want query error")
	}
	if _, err := GetSongs(db); err == nil {
		t.Fatal("GetSongs() error = nil, want query error")
	}
	if _, err := GetEpisodes(db); err == nil {
		t.Fatal("GetEpisodes() error = nil, want query error")
	}
}

func TestMediaDBScanErrors(t *testing.T) {
	t.Run("movies", func(t *testing.T) {
		db := newMovieScanErrorDB(t)
		if _, err := GetMovies(db); err == nil {
			t.Fatal("GetMovies() error = nil, want scan error")
		}
	})

	t.Run("songs", func(t *testing.T) {
		db := newSongScanErrorDB(t)
		if _, err := GetSongs(db); err == nil {
			t.Fatal("GetSongs() error = nil, want scan error")
		}
	})

	t.Run("episodes", func(t *testing.T) {
		db := newEpisodeScanErrorDB(t)
		if _, err := GetEpisodes(db); err == nil {
			t.Fatal("GetEpisodes() error = nil, want scan error")
		}
	})
}

func newMovieScanErrorDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openStructsTestDB(t)
	execStatements(t, db,
		`CREATE TABLE metadata_items (
			id INTEGER PRIMARY KEY, title TEXT, content_rating TEXT, year TEXT,
			tags_genre TEXT, rating REAL, audience_rating REAL, hash TEXT
		);`,
		`CREATE TABLE library_sections (id INTEGER PRIMARY KEY, name TEXT, section_type INTEGER);`,
		`CREATE TABLE media_items (
			id INTEGER PRIMARY KEY, metadata_item_id INTEGER, library_section_id INTEGER,
			container TEXT, bitrate INTEGER, video_codec TEXT, height INTEGER, width INTEGER,
			audio_codec TEXT
		);`,
		`CREATE TABLE media_parts (
			id INTEGER PRIMARY KEY, media_item_id INTEGER, file TEXT, hash TEXT, size INTEGER,
			duration INTEGER
		);`,
		`INSERT INTO library_sections VALUES (1, 'Movies', 1);`,
		`INSERT INTO metadata_items VALUES (1, 'Alien', 'R', 'bad', 'Horror', 8.5, 9.1, 'metadata');`,
		`INSERT INTO media_items VALUES (1, 1, 1, 'mkv', 5000, 'h264', 1080, 1920, 'aac');`,
		`INSERT INTO media_parts VALUES (1, 1, '/media/alien.mkv', 'file', 1, 1);`,
	)
	return db
}

func newSongScanErrorDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openStructsTestDB(t)
	execStatements(t, db,
		`CREATE TABLE metadata_items (
			id INTEGER PRIMARY KEY, parent_id INTEGER, metadata_type INTEGER, title TEXT,
			year TEXT, tags_genre TEXT, hash TEXT
		);`,
		`CREATE TABLE library_sections (id INTEGER PRIMARY KEY, name TEXT, section_type INTEGER);`,
		`CREATE TABLE media_items (
			id INTEGER PRIMARY KEY, metadata_item_id INTEGER, library_section_id INTEGER,
			bitrate INTEGER, audio_codec TEXT
		);`,
		`CREATE TABLE media_parts (
			id INTEGER PRIMARY KEY, media_item_id INTEGER, file TEXT, hash TEXT, size INTEGER,
			duration INTEGER
		);`,
		`INSERT INTO library_sections VALUES (1, 'Music', 8);`,
		`INSERT INTO metadata_items VALUES (1, NULL, 8, 'Artist', 1990, 'Rock', 'artist');`,
		`INSERT INTO metadata_items VALUES (2, 1, 9, 'Album', 'bad', 'Rock', 'album');`,
		`INSERT INTO metadata_items VALUES (3, 2, 10, 'Song', 'bad', 'Rock', 'song');`,
		`INSERT INTO media_items VALUES (1, 3, 1, 900, 'flac');`,
		`INSERT INTO media_parts VALUES (1, 1, '/media/song.flac', 'file', 1, 1);`,
	)
	return db
}

func newEpisodeScanErrorDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openStructsTestDB(t)
	execStatements(t, db,
		`CREATE TABLE metadata_items (
			id INTEGER PRIMARY KEY, parent_id INTEGER, metadata_type INTEGER, "index" TEXT,
			title TEXT, content_rating TEXT, year INTEGER, rating REAL, audience_rating REAL,
			hash TEXT
		);`,
		`CREATE TABLE library_sections (id INTEGER PRIMARY KEY, name TEXT, section_type INTEGER);`,
		`CREATE TABLE media_items (
			id INTEGER PRIMARY KEY, metadata_item_id INTEGER, library_section_id INTEGER,
			container TEXT, bitrate INTEGER, video_codec TEXT, height INTEGER, width INTEGER,
			audio_codec TEXT
		);`,
		`CREATE TABLE media_parts (
			id INTEGER PRIMARY KEY, media_item_id INTEGER, file TEXT, hash TEXT, size INTEGER,
			duration INTEGER
		);`,
		`INSERT INTO library_sections VALUES (1, 'TV', 2);`,
		`INSERT INTO metadata_items VALUES (1, NULL, 2, NULL, 'Show', NULL, 2020, 8.0, 8.5, 'show');`,
		`INSERT INTO metadata_items VALUES (2, 1, 3, 'bad', 'Season 1', NULL, 2020, NULL, NULL, 'season');`,
		`INSERT INTO metadata_items VALUES (3, 2, 4, 2, 'Episode', 'TV-14', 2020, NULL, NULL, 'episode');`,
		`INSERT INTO media_items VALUES (1, 3, 1, 'mkv', 4000, 'h264', 720, 1280, 'aac');`,
		`INSERT INTO media_parts VALUES (1, 1, '/media/show.mkv', 'file', 1, 1);`,
	)
	return db
}

func openStructsTestDB(t *testing.T) *sql.DB {
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
	return db
}

func execStatements(t *testing.T, db *sql.DB, statements ...string) {
	t.Helper()
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("Exec(%q) error = %v", stmt, err)
		}
	}
}
