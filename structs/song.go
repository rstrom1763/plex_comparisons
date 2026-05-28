package structs

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/rstrom1763/plex_comparisons/constants"
)

type Song struct {
	Title         string `json:"title"`         // metadata_items.title
	ContentRating string `json:"rating"`        // metadata_items.content_rating
	Year          int    `json:"year"`          // metadata_items.year
	Genre         string `json:"genre"`         // metadata_items.tags_genre (nullable)
	AlbumTitle    string `json:"album_title"`   // album_title
	ArtistName    string `json:"artist_name"`   // artist_name
	Library       string `json:"library"`       // library_sections.name
	MediaType     string `json:"media_type"`    // derived from library_sections.section_type
	File          string `json:"file"`          // media_parts.file
	Hash          string `json:"hash"`          // media_parts.hash
	Size          int64  `json:"size"`          // media_parts.size
	Duration      int64  `json:"duration"`      // media_parts.duration
	Container     string `json:"container"`     // media_items.container
	Bitrate       int    `json:"bitrate"`       // media_items.bitrate
	VideoCodec    string `json:"video_codec"`   // media_items.video_codec
	Height        int    `json:"height"`        // media_items.height
	Width         int    `json:"width"`         // media_items.width
	Resolution    string `json:"resolution"`    // derived based on width
	AudioCodec    string `json:"audio_codec"`   // media_items.audio_codec
	MetadataHash  string `json:"metadata_hash"` // metadata_items.hash
}

func (s *Song) GetTitle() string {
	return s.Title
}

func (s *Song) GetYear() int {
	return s.Year
}

func (s *Song) GetSizeBytes() int64 {
	return s.Size
}

func (s *Song) ToCSV() string {
	fields := []string{
		s.Title,
		strconv.Itoa(s.Year),
		s.Genre,
		s.AlbumTitle,
		s.ArtistName,
		s.Library,
		s.MediaType,
		s.File,
		s.Hash,
		strconv.FormatInt(s.Size, 10),
		strconv.FormatInt(s.Duration, 10),
		strconv.Itoa(s.Bitrate),
		s.AudioCodec,
		s.MetadataHash,
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

func (s *Song) CSVHeaders() string {
	return "title,year,genre,album_title,artist_name,library,media_type,file,hash,size,duration,bitrate,audio_codec,metadata_hash\n"
}

func GetSongs(db *sql.DB) ([]*Song, error) {
	rows, err := db.Query(SONG_DUMP_QUERY)
	if err != nil {
		return nil, fmt.Errorf("could not query Songs: %w", err)
	}

	var (
		Title        string
		Year         int
		Genre        string
		AlbumTitle   string
		ArtistName   string
		Library      string
		MediaType    string
		File         string
		Hash         string
		Size         int64
		Duration     int64
		Bitrate      int
		AudioCodec   string
		MetadataHash string
	)
	var Songs []*Song

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		err = rows.Scan(&Title, &Year, &Genre, &AlbumTitle, &ArtistName, &Library, &MediaType, &File, &Hash, &Size, &Duration, &Bitrate, &AudioCodec, &MetadataHash)
		if err != nil {
			return nil, fmt.Errorf("there was an error scanning the row: %w", err)
		}

		Song := Song{
			Title:        Title,
			Year:         Year,
			Genre:        Genre,
			AlbumTitle:   AlbumTitle,
			ArtistName:   ArtistName,
			Library:      Library,
			MediaType:    MediaType,
			File:         File,
			Hash:         Hash,
			Size:         Size,
			Duration:     Duration,
			Bitrate:      Bitrate,
			AudioCodec:   AudioCodec,
			MetadataHash: MetadataHash,
		}

		Songs = append(Songs, &Song)

	}
	return Songs, nil
}

func GetSongsFromCSVFile(path string) ([]*Song, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open csv file")
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	r := csv.NewReader(file)
	// We expect exactly 14 columns
	r.FieldsPerRecord = 14

	var songs []*Song
	row := 0

	for {
		record, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("csv read error on row %d: %w", row, err)
		}
		row++

		// Skip header row if present
		if row == 1 && len(record) == 14 &&
			strings.EqualFold(strings.TrimSpace(record[0]), "title") &&
			strings.EqualFold(strings.TrimSpace(record[1]), "year") {
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

		year, err := atoi(record[1])
		if err != nil {
			return nil, fmt.Errorf("invalid year on row %d: %w", row, err)
		}
		size, err := atoi64(record[9])
		if err != nil {
			return nil, fmt.Errorf("invalid size on row %d: %w", row, err)
		}
		duration, err := atoi64(record[10])
		if err != nil {
			return nil, fmt.Errorf("invalid duration on row %d: %w", row, err)
		}
		bitrate, err := atoi(record[11])
		if err != nil {
			return nil, fmt.Errorf("invalid bitrate on row %d: %w", row, err)
		}

		s := &Song{
			Title:        trim(record[0]),
			Year:         year,
			Genre:        trim(record[2]),
			AlbumTitle:   trim(record[3]),
			ArtistName:   trim(record[4]),
			Library:      trim(record[5]),
			MediaType:    trim(record[6]),
			File:         trim(record[7]),
			Hash:         trim(record[8]),
			Size:         size,
			Duration:     duration,
			Bitrate:      bitrate,
			AudioCodec:   trim(record[12]),
			MetadataHash: trim(record[13]),
		}

		songs = append(songs, s)
	}

	return songs, nil
}

// GetUniqueTitle Aggregates Title and Year to avoid issues with songs with same title
func (s *Song) GetUniqueTitle() string {
	uniqueTitle := fmt.Sprintf("%s %s %s %s", s.Title, s.ArtistName, s.AlbumTitle, strconv.Itoa(s.Year))
	return uniqueTitle
}
