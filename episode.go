package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type Episode struct {
	ShowTitle     string `json:"show_title"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	EpisodeTitle  string `json:"episode_title"`
	ContentRating string `json:"rating"`
	Year          int    `json:"year"`
	Library       string `json:"library"`
	MediaType     string `json:"media_type"`
	File          string `json:"file"`
	Hash          string `json:"hash"`
	Size          int64  `json:"size"`
	Duration      int64  `json:"duration"`
	Container     string `json:"container"`
	Bitrate       int    `json:"bitrate"`
	VideoCodec    string `json:"video_codec"`
	Height        int    `json:"height"`
	Width         int    `json:"width"`
	Resolution    string `json:"resolution"`
	AudioCodec    string `json:"audio_codec"`
}

func (e *Episode) GetTitle() string {
	return e.EpisodeTitle
}

func (e *Episode) GetYear() int {
	return e.Year
}

func (e *Episode) ToCSV() string {
	fields := []string{
		e.ShowTitle,
		strconv.Itoa(e.SeasonNumber),
		strconv.Itoa(e.EpisodeNumber),
		e.EpisodeTitle,
		e.ContentRating,
		strconv.Itoa(e.Year),
		e.Library,
		e.MediaType,
		e.File,
		e.Hash,
		strconv.FormatInt(e.Size, 10),
		strconv.FormatInt(e.Duration, 10),
		e.Container,
		strconv.Itoa(e.Bitrate),
		e.VideoCodec,
		strconv.Itoa(e.Height),
		strconv.Itoa(e.Width),
		e.Resolution,
		e.AudioCodec,
	}

	for i, field := range fields {
		if strings.ContainsAny(field, `",`) {
			field = strings.ReplaceAll(field, `"`, `""`)
			fields[i] = `"` + field + `"`
		}
	}

	return strings.Join(fields, ",") + "\n"
}

func (e *Episode) CSVHeaders() string {
	return "show_title,season_number,episode_number,episode_title,rating,year,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec\n"
}

func getEpisodes(db *sql.DB) ([]*Episode, error) {
	rows, err := db.Query(EPISODE_DUMP_QUERY)
	if err != nil {
		return nil, fmt.Errorf("could not query Episodes: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing DB rows: %s", err)
		}
	}()

	var episodes []*Episode
	for rows.Next() {
		var (
			ShowTitle     string
			SeasonNumber  int
			EpisodeNumber int
			EpisodeTitle  string
			ContentRating string
			Year          int
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

		err = rows.Scan(
			&ShowTitle, &SeasonNumber, &EpisodeNumber, &EpisodeTitle, &ContentRating,
			&Year, &Library, &MediaType, &File, &Hash, &Size, &Duration,
			&Container, &Bitrate, &VideoCodec, &Height, &Width, &Resolution, &AudioCodec,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		e := &Episode{
			ShowTitle:     ShowTitle,
			SeasonNumber:  SeasonNumber,
			EpisodeNumber: EpisodeNumber,
			EpisodeTitle:  EpisodeTitle,
			ContentRating: ContentRating,
			Year:          Year,
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
		episodes = append(episodes, e)
	}

	return episodes, nil
}

func getEpisodesFromCSVFile(path string) ([]*Episode, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open csv file")
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("could not close csv file: %v", path)
		}
	}()

	r := csv.NewReader(f)
	// Expect exactly 19 columns (see Episode.CSVHeaders / Episode.ToCSV)
	r.FieldsPerRecord = 19

	var episodes []*Episode
	row := 0

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

	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("csv read error on row %d: %w", row, err)
		}
		row++

		// Skip header row if present
		if row == 1 && len(rec) == 19 &&
			strings.EqualFold(trim(rec[0]), "show_title") &&
			strings.EqualFold(trim(rec[1]), "season_number") &&
			strings.EqualFold(trim(rec[2]), "episode_number") {
			continue
		}

		season, err := atoi(rec[1])
		if err != nil {
			return nil, fmt.Errorf("invalid season_number on row %d: %w", row, err)
		}
		epNum, err := atoi(rec[2])
		if err != nil {
			return nil, fmt.Errorf("invalid episode_number on row %d: %w", row, err)
		}
		year, err := atoi(rec[5])
		if err != nil {
			return nil, fmt.Errorf("invalid year on row %d: %w", row, err)
		}
		size, err := atoi64(rec[10])
		if err != nil {
			return nil, fmt.Errorf("invalid size on row %d: %w", row, err)
		}
		duration, err := atoi64(rec[11])
		if err != nil {
			return nil, fmt.Errorf("invalid duration on row %d: %w", row, err)
		}
		bitrate, err := atoi(rec[13])
		if err != nil {
			return nil, fmt.Errorf("invalid bitrate on row %d: %w", row, err)
		}
		height, err := atoi(rec[15])
		if err != nil {
			return nil, fmt.Errorf("invalid height on row %d: %w", row, err)
		}
		width, err := atoi(rec[16])
		if err != nil {
			return nil, fmt.Errorf("invalid width on row %d: %w", row, err)
		}

		e := &Episode{
			ShowTitle:     trim(rec[0]),
			SeasonNumber:  season,
			EpisodeNumber: epNum,
			EpisodeTitle:  trim(rec[3]),
			ContentRating: trim(rec[4]),
			Year:          year,
			Library:       trim(rec[6]),
			MediaType:     trim(rec[7]),
			File:          trim(rec[8]),
			Hash:          trim(rec[9]),
			Size:          size,
			Duration:      duration,
			Container:     trim(rec[12]),
			Bitrate:       bitrate,
			VideoCodec:    trim(rec[14]),
			Height:        height,
			Width:         width,
			Resolution:    trim(rec[17]),
			AudioCodec:    trim(rec[18]),
		}

		episodes = append(episodes, e)
	}

	return episodes, nil
}

// GetUniqueTitle Aggregates ShowTitle, EpisodeTitle and Year to avoid issues with episodes with same title
func (e *Episode) GetUniqueTitle() string {
	uniqueTitle := fmt.Sprintf("%s %s %d", e.ShowTitle, e.EpisodeTitle, e.Year)
	return uniqueTitle
}
