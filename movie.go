package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Movie struct {
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

func (m *Movie) GetTitle() string {
	return m.Title
}

func (m *Movie) GetYear() int {
	return m.Year
}

func (m *Movie) ToCSV() string {
	fields := []string{
		m.Title,
		m.ContentRating,
		strconv.Itoa(m.Year),
		m.Genre,
		m.Library,
		m.MediaType,
		m.File,
		m.Hash,
		strconv.FormatInt(m.Size, 10),
		strconv.FormatInt(m.Duration, 10),
		m.Container,
		strconv.Itoa(m.Bitrate),
		m.VideoCodec,
		strconv.Itoa(m.Height),
		strconv.Itoa(m.Width),
		m.Resolution,
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

func (m *Movie) CSVHeaders() string {
	return "title,rating,year,genre,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec\n"
}

func getMovies(db *sql.DB) ([]*Movie, error) {

	rows, err := db.Query(MOVIE_DUMP_QUERY)
	if err != nil {
		return nil, fmt.Errorf("could not query Movies: " + err.Error())
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
	var movies []*Movie

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing DB rows: %s", err)
		}
	}(rows)

	for rows.Next() {
		err = rows.Scan(&Title, &ContentRating, &Year, &Genre, &Library, &MediaType, &File, &Hash, &Size, &Duration, &Container, &Bitrate, &VideoCodec, &Height, &Width, &Resolution, &AudioCodec)
		if err != nil {
			return nil, fmt.Errorf("there was an error scanning the row: " + err.Error())
		}

		movie := Movie{
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

		movies = append(movies, &movie)

	}
	return movies, nil
}

func getMoviesFromCSVFile(path string) ([]*Movie, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open csv file")
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("could not close csv file: %v", path)
		}
	}(file)

	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = 17

	var movies []*Movie
	row := 0

	for {
		record, err := csvReader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("csv read error on row %d: %w", row, err)
		}
		row++

		// Skip header row
		if row == 1 && len(record) == 17 &&
			strings.EqualFold(strings.TrimSpace(record[0]), "title") &&
			strings.EqualFold(strings.TrimSpace(record[1]), "rating") {
			continue
		}

		trim := func(s string) string { return strings.TrimSpace(s) }

		atoi := func(s string) (int, error) {
			if s = trim(s); s == "" {
				return 0, nil
			}
			return strconv.Atoi(s)
		}
		atoi64 := func(s string) (int64, error) {
			if s = trim(s); s == "" {
				return 0, nil
			}
			return strconv.ParseInt(s, 10, 64)
		}

		year, err := atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid year on row %d: %w", row, err)
		}
		size, err := atoi64(record[8])
		if err != nil {
			return nil, fmt.Errorf("invalid size on row %d: %w", row, err)
		}
		duration, err := atoi64(record[9])
		if err != nil {
			return nil, fmt.Errorf("invalid duration on row %d: %w", row, err)
		}
		bitrate, err := atoi(record[11])
		if err != nil {
			return nil, fmt.Errorf("invalid bitrate on row %d: %w", row, err)
		}
		height, err := atoi(record[13])
		if err != nil {
			return nil, fmt.Errorf("invalid height on row %d: %w", row, err)
		}
		width, err := atoi(record[14])
		if err != nil {
			return nil, fmt.Errorf("invalid width on row %d: %w", row, err)
		}

		movie := &Movie{
			Title:         trim(record[0]),
			ContentRating: trim(record[1]),
			Year:          year,
			Genre:         trim(record[3]),
			Library:       trim(record[4]),
			MediaType:     trim(record[5]),
			File:          trim(record[6]),
			Hash:          trim(record[7]),
			Size:          size,
			Duration:      duration,
			Container:     trim(record[10]),
			Bitrate:       bitrate,
			VideoCodec:    trim(record[12]),
			Height:        height,
			Width:         width,
			Resolution:    trim(record[15]),
			AudioCodec:    trim(record[16]),
		}

		movies = append(movies, movie)
	}

	return movies, nil
}

// GetUniqueTitle Aggregates Title and Year to avoid issues with movies with same title
func (m *Movie) GetUniqueTitle() string {
	uniqueTitle := fmt.Sprintf("%s %s", m.Title, strconv.Itoa(m.Year))
	return uniqueTitle
}
