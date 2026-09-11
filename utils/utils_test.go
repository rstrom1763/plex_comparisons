package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddNoHaveToPath(t *testing.T) {
	got := AddNoHaveToPath(filepath.Join("exports", "movies.csv"))
	want := filepath.Join("exports", "movies_no_have.csv")
	if got != want {
		t.Fatalf("AddNoHaveToPath() = %q, want %q", got, want)
	}
}

func TestInitDBOpensSQLiteDatabase(t *testing.T) {
	db, err := InitDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
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

	got, err := Env("UNIT_TEST_ENV")
	if err != nil {
		t.Fatalf("Env() error = %v", err)
	}
	if got != "value" {
		t.Fatalf("Env() = %q, want %q", got, "value")
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

	if _, err := Env("MISSING"); err == nil {
		t.Fatal("Env() error = nil, want error")
	}
}

func TestReplacePathPrefix(t *testing.T) {
	for _, tt := range []struct{ name, path, from, to, want string }{
		{"prefix", "/movies/Alien.mkv", "/movies/", "/mnt/movies/", "/mnt/movies/Alien.mkv"},
		{"missing source", "/movies/Alien.mkv", "", "/mnt/", "/movies/Alien.mkv"},
		{"missing target", "/movies/Alien.mkv", "/movies/", "", "/movies/Alien.mkv"},
		{"no match", "/other/Alien.mkv", "/movies/", "/mnt/", "/other/Alien.mkv"},
		{"interior match", "/other/movies/Alien.mkv", "/movies/", "/mnt/", "/other/movies/Alien.mkv"},
		{"repeated prefix", "/movies/movies/Alien.mkv", "/movies/", "/mnt/", "/mnt/movies/Alien.mkv"},
		{"case sensitive", "/Movies/Alien.mkv", "/movies/", "/mnt/", "/Movies/Alien.mkv"},
		{"no separator conversion", "/movies/sub/Alien.mkv", "/movies/", `D:\Movies\`, `D:\Movies\sub/Alien.mkv`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplacePathPrefix(tt.path, tt.from, tt.to); got != tt.want {
				t.Fatalf("ReplacePathPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
