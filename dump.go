package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func dump(plexDbPath string) error {

	db, err := initDB(plexDbPath)
	if err != nil {
		return fmt.Errorf("there was an error initializing the DB connection: " + err.Error())
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("could not close Plex database: ", err)
		}
	}(db)

	movies, err := getMovies(db)
	if err != nil {
		return fmt.Errorf("could not fetch movies from DB: %v", err.Error())
	}

	err = writeCSV("./movies.csv", movies)
	if err != nil {
		return fmt.Errorf("error writing movies dump: " + err.Error())
	}

	songs, err := getSongs(db)
	if err != nil {
		return fmt.Errorf("could not fetch songs from DB: %v", err.Error())
	}

	err = writeCSV("./songs.csv", songs)
	if err != nil {
		return fmt.Errorf("error writing songs dump: " + err.Error())
	}

	episodes, err := getEpisodes(db)
	if err != nil {
		return fmt.Errorf("could not fetch episodes from DB: %v", err.Error())
	}

	err = writeCSV("./episodes.csv", episodes)
	if err != nil {
		return fmt.Errorf("error writing episodes dump: " + err.Error())
	}

	return nil

}

func writeCSV[T Media](filename string, data []T) error {

	if len(data) == 0 {
		return fmt.Errorf("input is empty")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: " + err.Error())
	}

	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Println("could not close file handle: " + err.Error())
		}
	}(file)

	var lines []string

	// Add header
	lines = append(lines, data[0].CSVHeaders())

	// Add records
	for _, record := range data {
		lines = append(lines, record.ToCSV())
	}

	// Write data to file
	_, err = file.Write([]byte(strings.Join(lines, "")))
	if err != nil {
		return fmt.Errorf("could not write dump files: %v", err)
	}

	return nil

}
