package main

import (
	"strconv"
	"strings"
)

type Media interface {
	ToCSV() string
	CSVHeaders() string
}

type MediaItem struct {
	Title         string `json:"title"`  // metadata_items.title
	ContentRating string `json:"rating"` // metadata_items.content_rating
	Year          int    `json:"year"`   // metadata_items.year
	Genre         string `json:"genre"`  // metadata_items.tags_genre (nullable)

	Library   string `json:"library"`    // library_sections.name
	MediaType string `json:"media_type"` // derived from library_sections.section_type

	File     string `json:"file"`     // media_parts.file
	Hash     string `json:"hash"`     // media_parts.hash
	Size     int64  `json:"size"`     // media_parts.size
	Duration int64  `json:"duration"` // media_parts.duration (usually in ms)

	Container  string `json:"container"`   // media_items.container
	Bitrate    int    `json:"bitrate"`     // media_items.bitrate
	VideoCodec string `json:"video_codec"` // media_items.video_codec
	Height     int    `json:"height"`      // media_items.height
	Width      int    `json:"width"`       // media_items.width
	Resolution string `json:"resolution"`  // derived based on width

	AudioCodec string `json:"audio_codec"` // media_items.audio_codec
}

func (m *MediaItem) ToCSV() string {
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

func (m *MediaItem) CSVHeaders() string {
	return "title,rating,year,genre,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec\n"
}
