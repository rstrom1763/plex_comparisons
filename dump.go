package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func dump(path string) error {

	db, err := initDB(PLEX_DB_PATH)
	if err != nil {
		return fmt.Errorf("there was an error: " + err.Error())
	}

	rows, err := db.Query(DUMP_QUERY)
	if err != nil {
		return fmt.Errorf("could not query DB: " + err.Error())
	}

	var (
		Title         string
		ContentRating string
		Year          int
		Genre         string
		Library       string
		MediaType     string
		File          string
		Hash          string
		Size          int64
		Duration      int64
		Container     string
		Bitrate       int
		VideoCodec    string
		Height        int
		Width         int
		Resolution    string
		AudioCodec    string
	)
	var mediaItems []MediaItem

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing DB rows: %s", err)
		}
	}(rows)

	for rows.Next() {
		err = rows.Scan(&Title, &ContentRating, &Year, &Genre, &Library, &MediaType, &File, &Hash, &Size, &Duration, &Container, &Bitrate, &VideoCodec, &Height, &Width, &Resolution, &AudioCodec)
		if err != nil {
			return fmt.Errorf("there was an error scanning the row: " + err.Error())
		}

		mediaItem := MediaItem{
			Title:         Title,
			ContentRating: ContentRating,
			Year:          Year,
			Genre:         Genre,
			Library:       Library,
			MediaType:     MediaType,
			File:          File,
			Hash:          Hash,
			Size:          Size,
			Duration:      Duration,
			Container:     Container,
			Bitrate:       Bitrate,
			VideoCodec:    VideoCodec,
			Height:        Height,
			Width:         Width,
			Resolution:    Resolution,
			AudioCodec:    AudioCodec,
		}

		mediaItems = append(mediaItems, mediaItem)

	}

	err = writeDump(path, mediaItems)
	if err != nil {
		return fmt.Errorf("error writing dump: " + err.Error())
	}

	return nil

}
