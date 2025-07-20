package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Song struct {
	Title         string `json:"title"`       // metadata_items.title
	ContentRating string `json:"rating"`      // metadata_items.content_rating
	Year          int    `json:"year"`        // metadata_items.year
	Genre         string `json:"genre"`       // metadata_items.tags_genre (nullable)
	Library       string `json:"library"`     // library_sections.name
	MediaType     string `json:"media_type"`  // derived from library_sections.section_type
	File          string `json:"file"`        // media_parts.file
	Hash          string `json:"hash"`        // media_parts.hash
	Size          int64  `json:"size"`        // media_parts.size
	Duration      int64  `json:"duration"`    // media_parts.duration
	Container     string `json:"container"`   // media_items.container
	Bitrate       int    `json:"bitrate"`     // media_items.bitrate
	VideoCodec    string `json:"video_codec"` // media_items.video_codec
	Height        int    `json:"height"`      // media_items.height
	Width         int    `json:"width"`       // media_items.width
	Resolution    string `json:"resolution"`  // derived based on width
	AudioCodec    string `json:"audio_codec"` // media_items.audio_codec
}

func (m *Song) GetTitle() string {
	return m.Title
}

func (m *Song) GetYear() int {
	return m.Year
}

func (m *Song) ToCSV() string {
	fields := []string{
		m.Title,
		strconv.Itoa(m.Year),
		m.Genre,
		m.Library,
		m.MediaType,
		m.File,
		m.Hash,
		strconv.FormatInt(m.Size, 10),
		strconv.FormatInt(m.Duration, 10),
		strconv.Itoa(m.Bitrate),
		m.AudioCodec,
	}

	// Escape commas or quotes by wrapping each field in quotes if necessary
	for i, field := range fields {
		if strings.ContainsAny(field, `",`) {
			field = strings.ReplaceAll(field, `"`, `""`) // escape quotes
			fields[i] = `"` + field + `"`
		}
	}

	final := strings.Join(fields, ",")
	final += "\n"

	return final
}

func (m *Song) CSVHeaders() string {
	return "title,year,genre,library,media_type,file,hash,size,duration,bitrate,audio_codec\n"
}

func getSongs(db *sql.DB) ([]*Song, error) {

	rows, err := db.Query(SONG_DUMP_QUERY)
	if err != nil {
		return nil, fmt.Errorf("could not query Songs: " + err.Error())
	}

	var (
		Title      string
		Year       int
		Genre      string
		Library    string
		MediaType  string
		File       string
		Hash       string
		Size       int64
		Duration   int64
		Bitrate    int
		AudioCodec string
	)
	var Songs []*Song

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing DB rows: %s", err)
		}
	}(rows)

	for rows.Next() {
		err = rows.Scan(&Title, &Year, &Genre, &Library, &MediaType, &File, &Hash, &Size, &Duration, &Bitrate, &AudioCodec)
		if err != nil {
			return nil, fmt.Errorf("there was an error scanning the row: " + err.Error())
		}

		Song := Song{
			Title:      Title,
			Year:       Year,
			Genre:      Genre,
			Library:    Library,
			MediaType:  MediaType,
			File:       File,
			Hash:       Hash,
			Size:       Size,
			Duration:   Duration,
			Bitrate:    Bitrate,
			AudioCodec: AudioCodec,
		}

		Songs = append(Songs, &Song)

	}
	return Songs, nil
}
