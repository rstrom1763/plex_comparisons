package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	utils "github.com/rstrom1763/goUtils"
	. "github.com/rstrom1763/plex_comparisons/constants"
)

// Create the DB connection
func initDB(path string) (*sql.DB, error) {

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
func env(key string) (string, error) {

	// load .env file
	err := godotenv.Load(DOTENV_PATH)
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %s", err)
	}

	return os.Getenv(key), nil
}

func addNoHaveToPath(path string) string {
	prefix := path[:strings.LastIndex(path, ".")]
	fileExtension := path[strings.LastIndex(path, "."):]
	return prefix + "_no_have" + fileExtension
}

func getByteSumFromDumpFile(dumpPath string, mediaType string) (int64, error) {
	var byteSum int64

	if !utils.FileExists(dumpPath) {
		return 0, fmt.Errorf("%s does not exist", dumpPath)
	}

	items, err := getMediaItemsFromCSV(dumpPath, mediaType)
	if err != nil {
		return 0, fmt.Errorf("could not get media items from csv: %s", err.Error())
	}

	for _, item := range items {
		byteSum += item.GetSizeBytes()
	}

	return byteSum, nil

}
