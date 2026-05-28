package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/rstrom1763/plex_comparisons/structs"
)

func dump(plexDbPath string) error {

	db, err := initDB(plexDbPath)
	if err != nil {
		return fmt.Errorf("there was an error initializing the DB connection: %w", err)
	}

	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	movies, err := GetMovies(db)
	if err != nil {
		return fmt.Errorf("could not fetch movies from DB: %v", err.Error())
	}

	err = writeCSV("./movies.csv", movies)
	if err != nil {
		return fmt.Errorf("error writing movies dump: %w", err)
	}

	songs, err := GetSongs(db)
	if err != nil {
		return fmt.Errorf("could not fetch songs from DB: %v", err.Error())
	}

	err = writeCSV("./songs.csv", songs)
	if err != nil {
		return fmt.Errorf("error writing songs dump: %w", err)
	}

	episodes, err := GetEpisodes(db)
	if err != nil {
		return fmt.Errorf("could not fetch episodes from DB: %v", err.Error())
	}

	err = writeCSV("./episodes.csv", episodes)
	if err != nil {
		return fmt.Errorf("error writing episodes dump: %w", err)
	}

	return nil

}

func writeCSV[T Media](filename string, data []T) error {

	if len(data) == 0 {
		return nil
	}

	var lines []string

	// Add header
	lines = append(lines, data[0].CSVHeaders())

	// Add records
	for _, record := range data {
		lines = append(lines, record.ToCSV())
	}

	// Write data to file
	if err := os.WriteFile(filename, []byte(strings.Join(lines, "")), 0644); err != nil {
		return fmt.Errorf("could not write dump files: %v", err)
	}

	return nil

}
