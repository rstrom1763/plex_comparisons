package utils

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/rstrom1763/plex_comparisons/constants"
)

// Create the DB connection
func InitDB(path string) (*sql.DB, error) {

	// Create the db connection
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open DB: %s", err)
	}

	// Test if we can access the db properly
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not ping DB: %s", err)
	}

	return db, nil

}

// Get key from the env file
func Env(key string) (string, error) {

	// load .env file
	err := godotenv.Load(DOTENV_PATH)
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %s", err)
	}

	return os.Getenv(key), nil
}

func AddNoHaveToPath(path string) string {
	prefix := path[:strings.LastIndex(path, ".")]
	fileExtension := path[strings.LastIndex(path, "."):]
	return prefix + "_no_have" + fileExtension
}

// ReplacePathPrefix replaces a literal, case-sensitive prefix without converting
// separators. Missing mappings and paths that do not match are left unchanged.
func ReplacePathPrefix(path, from, to string) string {
	if from == "" || to == "" || !strings.HasPrefix(path, from) {
		return path
	}
	return to + strings.TrimPrefix(path, from)
}
