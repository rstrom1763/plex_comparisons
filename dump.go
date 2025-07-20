package main

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

func dump() error {

	db, err := initDB(PLEX_DB_PATH)
	if err != nil {
		return fmt.Errorf("there was an error initializing the DB connection: " + err.Error())
	}

	movies, err := getMovies(db)
	if err != nil {
		return fmt.Errorf("could not fetch movies from DB: %v", err.Error())
	}

	err = writeDump("./movies.csv", movies)
	if err != nil {
		return fmt.Errorf("error writing movies dump: " + err.Error())
	}

	songs, err := getSongs(db)
	if err != nil {
		return fmt.Errorf("could not fetch songs from DB: %v", err.Error())
	}

	err = writeDump("./songs.csv", songs)
	if err != nil {
		return fmt.Errorf("error writing songs dump: " + err.Error())
	}

	episodes, err := getEpisodes(db)
	if err != nil {
		return fmt.Errorf("could not fetch episodes from DB: %v", err.Error())
	}

	err = writeDump("./episodes.csv", episodes)
	if err != nil {
		return fmt.Errorf("error writing epidodes dump: " + err.Error())
	}

	return nil

}

func writeDump[T Media](filename string, data []T) error {

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

	// Write header
	_, err = file.Write([]byte(data[0].CSVHeaders()))
	if err != nil {
		return fmt.Errorf("could not write header: " + err.Error())
	}

	// Write records
	for _, record := range data {
		_, err = file.Write([]byte(record.ToCSV()))
		if err != nil {
			return fmt.Errorf("could not write record to csv: " + err.Error())
		}
	}
	return nil
	
}
