package main

import (
	"database/sql"
	"fmt"
	"log"
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
