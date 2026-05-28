package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddNoHaveToPath(t *testing.T) {
	got := addNoHaveToPath(filepath.Join("exports", "movies.csv"))
	want := filepath.Join("exports", "movies_no_have.csv")
	if got != want {
		t.Fatalf("addNoHaveToPath() = %q, want %q", got, want)
	}
}

func TestInitDBOpensSQLiteDatabase(t *testing.T) {
	db, err := initDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initDB() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestEnvLoadsDotenvFile(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("UNIT_TEST_ENV=value\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(tempDir) error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Chdir(originalDir) error = %v", err)
		}
		os.Unsetenv("UNIT_TEST_ENV")
	})

	got, err := env("UNIT_TEST_ENV")
	if err != nil {
		t.Fatalf("env() error = %v", err)
	}
	if got != "value" {
		t.Fatalf("env() = %q, want %q", got, "value")
	}
}

func TestEnvReturnsLoadError(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(tempDir) error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Chdir(originalDir) error = %v", err)
		}
	})

	if _, err := env("MISSING"); err == nil {
		t.Fatal("env() error = nil, want error")
	}
}

func TestGetByteSumFromDumpFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "movies.csv")
	data := (&testByteSumMovie{}).CSVHeaders() +
		"Alien,,1979,,,,,,100,0,,0,,0,0,,,0.0,0.0,,0.00\n" +
		"Aliens,,1986,,,,,,200,0,,0,,0,0,,,0.0,0.0,,0.00\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := getByteSumFromDumpFile(path, "movie")
	if err != nil {
		t.Fatalf("getByteSumFromDumpFile() error = %v", err)
	}
	if got != 300 {
		t.Fatalf("getByteSumFromDumpFile() = %d, want 300", got)
	}
}

func TestGetByteSumFromDumpFileReturnsParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "movies.csv")
	if err := os.WriteFile(path, []byte("not,a,valid,movie,csv\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := getByteSumFromDumpFile(path, "movie"); err == nil {
		t.Fatal("getByteSumFromDumpFile() error = nil, want error")
	}
}

func TestGetByteSumFromDumpFileReturnsMissingFileError(t *testing.T) {
	if _, err := getByteSumFromDumpFile(filepath.Join(t.TempDir(), "missing.csv"), "movie"); err == nil {
		t.Fatal("getByteSumFromDumpFile() error = nil, want missing file error")
	}
}

type testByteSumMovie struct{}

func (m *testByteSumMovie) CSVHeaders() string {
	return "title,rating,year,genre,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec,critic_rating,audience_rating,metadata_hash,quality_score\n"
}
