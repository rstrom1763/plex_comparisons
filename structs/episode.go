package structs

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	. "github.com/rstrom1763/plex_comparisons/constants"
)

type Episode struct {
	ShowTitle      string  `json:"show_title"`
	SeasonNumber   int     `json:"season_number"`
	EpisodeNumber  int     `json:"episode_number"`
	EpisodeTitle   string  `json:"episode_title"`
	ContentRating  string  `json:"rating"`
	Year           int     `json:"year"`
	Library        string  `json:"library"`
	MediaType      string  `json:"media_type"`
	File           string  `json:"file"`
	Hash           string  `json:"hash"`
	Size           int64   `json:"size"`
	Duration       int64   `json:"duration"`
	Container      string  `json:"container"`
	Bitrate        int     `json:"bitrate"`
	VideoCodec     string  `json:"video_codec"`
	Height         int     `json:"height"`
	Width          int     `json:"width"`
	Resolution     string  `json:"resolution"`
	AudioCodec     string  `json:"audio_codec"`
	CriticRating   float64 `json:"critic_rating"`
	AudienceRating float64 `json:"audience_rating"`
	MetadataHash   string  `json:"metadata_hash"`
	QualityScore   float64 `json:"quality_score"` // computed field
}

func (e *Episode) GetTitle() string {
	return e.EpisodeTitle
}

func (e *Episode) GetYear() int {
	return e.Year
}

func (e *Episode) GetSizeBytes() int64 {
	return e.Size
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
		strconv.FormatFloat(e.CriticRating, 'f', 1, 64),
		strconv.FormatFloat(e.AudienceRating, 'f', 1, 64),
		e.MetadataHash,
		strconv.FormatFloat(e.QualityScore, 'f', 2, 64),
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
	return "show_title,season_number,episode_number,episode_title,rating,year,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec,critic_rating,audience_rating,metadata_hash,quality_score\n"
}

func (e *Episode) CalculateQualityScore() {
	// Base bitrate in kbps
	bitrate := float64(e.Bitrate)
	if bitrate <= 0 {
		e.QualityScore = 0
		return
	}

	// 1. Resolution Scaling with Diminishing Returns
	// Using a power function (exponent 0.7) to model that increasing resolution
	// requires more bitrate, but not linearly for perceived quality.
	sdPixels := 720.0 * 480.0
	currentPixels := float64(e.Width * e.Height)
	resolutionFactor := 1.0
	if currentPixels > 0 {
		resolutionFactor = math.Pow(currentPixels/sdPixels, 0.7)
	}

	// Calculate bitrate density (bits per relative pixel unit)
	score := bitrate / resolutionFactor

	// 2. Codec Efficiency Multipliers
	// HEVC is ~50% more efficient than H.264, AV1 is ~30% more efficient than HEVC.
	codecMultiplier := 1.0
	switch strings.ToLower(e.VideoCodec) {
	case "av1":
		codecMultiplier = 2.6 // Relative to H.264
	case "hevc", "h265", "x265":
		codecMultiplier = 2.0
	case "vp9":
		codecMultiplier = 1.8
	case "h264", "x264":
		codecMultiplier = 1.0
	case "vc1":
		codecMultiplier = 0.9
	case "mpeg4", "divx", "xvid":
		codecMultiplier = 0.7
	case "mpeg2video", "mpeg2":
		codecMultiplier = 0.6
	default:
		codecMultiplier = 1.0
	}
	score *= codecMultiplier

	// 3. Bit Depth Bonus
	if strings.Contains(strings.ToLower(e.VideoCodec), "10") {
		score *= 1.1
	}

	// 4. Audio Quality Component (Small weight)
	audioBonus := 1.0
	switch strings.ToLower(e.AudioCodec) {
	case "dca", "dts", "dts-hd", "truehd", "flac":
		audioBonus = 1.1
	case "ac3", "eac3", "aac":
		audioBonus = 1.05
	case "mp3", "vorbis", "opus":
		audioBonus = 1.0
	}
	score *= audioBonus

	// Final Scaling: Normalize to a range where a good 1080p HEVC encode
	// (e.g. 5Mbps) gets a score around 10.
	e.QualityScore = math.Round(score/300*100) / 100
}

func GetEpisodes(db *sql.DB) ([]*Episode, error) {
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
			ShowTitle      string
			SeasonNumber   int
			EpisodeNumber  int
			EpisodeTitle   string
			ContentRating  string
			Year           int
			Library        string
			MediaType      string
			File           string
			Hash           string
			Size           int64
			Duration       int64
			Container      string
			Bitrate        int
			VideoCodec     string
			Height         int
			Width          int
			Resolution     string
			AudioCodec     string
			CriticRating   float64
			AudienceRating float64
			MetadataHash   string
		)

		err = rows.Scan(
			&ShowTitle, &SeasonNumber, &EpisodeNumber, &EpisodeTitle, &ContentRating,
			&Year, &Library, &MediaType, &File, &Hash, &Size, &Duration,
			&Container, &Bitrate, &VideoCodec, &Height, &Width, &Resolution, &AudioCodec,
			&CriticRating, &AudienceRating, &MetadataHash,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		e := &Episode{
			ShowTitle:      ShowTitle,
			SeasonNumber:   SeasonNumber,
			EpisodeNumber:  EpisodeNumber,
			EpisodeTitle:   EpisodeTitle,
			ContentRating:  ContentRating,
			Year:           Year,
			Library:        Library,
			MediaType:      MediaType,
			File:           File,
			Hash:           Hash,
			Size:           Size,
			Duration:       Duration,
			Container:      Container,
			Bitrate:        Bitrate,
			VideoCodec:     VideoCodec,
			Height:         Height,
			Width:          Width,
			Resolution:     Resolution,
			AudioCodec:     AudioCodec,
			CriticRating:   math.Round(float64(CriticRating)*10) / 10,
			AudienceRating: math.Round(float64(AudienceRating)*10) / 10,
			MetadataHash:   MetadataHash,
		}
		e.CalculateQualityScore()
		episodes = append(episodes, e)
	}

	return episodes, nil
}

func GetEpisodesFromCSVFile(path string) ([]*Episode, error) {
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
	r.FieldsPerRecord = 23

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
	atof := func(s string) (float64, error) {
		if s = trim(s); s == "" {
			return 0, nil
		}
		return strconv.ParseFloat(s, 64)
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
		if row == 1 &&
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
		criticRating, err := atof(rec[19])
		if err != nil {
			return nil, fmt.Errorf("invalid critic_rating on row %d: %w", row, err)
		}
		audienceRating, err := atof(rec[20])
		if err != nil {
			return nil, fmt.Errorf("invalid audience_rating on row %d: %w", row, err)
		}

		qualityScore, err := atof(rec[22])
		if err != nil {
			return nil, fmt.Errorf("invalid quality_score on row %d: %w", row, err)
		}

		e := &Episode{
			ShowTitle:      trim(rec[0]),
			SeasonNumber:   season,
			EpisodeNumber:  epNum,
			EpisodeTitle:   trim(rec[3]),
			ContentRating:  trim(rec[4]),
			Year:           year,
			Library:        trim(rec[6]),
			MediaType:      trim(rec[7]),
			File:           trim(rec[8]),
			Hash:           trim(rec[9]),
			Size:           size,
			Duration:       duration,
			Container:      trim(rec[12]),
			Bitrate:        bitrate,
			VideoCodec:     trim(rec[14]),
			Height:         height,
			Width:          width,
			Resolution:     trim(rec[17]),
			AudioCodec:     trim(rec[18]),
			CriticRating:   math.Round(criticRating*10) / 10,
			AudienceRating: math.Round(audienceRating*10) / 10,
			MetadataHash:   trim(rec[21]),
			QualityScore:   qualityScore,
		}

		if len(rec) <= 22 {
			e.CalculateQualityScore()
		}

		episodes = append(episodes, e)
	}

	return episodes, nil
}

// GetUniqueTitle Aggregates ShowTitle, EpisodeTitle and Year to avoid issues with episodes with same title
func (e *Episode) GetUniqueTitle() string {
	uniqueTitle := fmt.Sprintf("%s %d %s %d", e.ShowTitle, e.SeasonNumber, e.EpisodeTitle, e.Year)
	return uniqueTitle
}

func (e *Episode) GetShowUniqueTitle() string {
	return fmt.Sprintf("%s %d", e.ShowTitle, e.Year)
}
