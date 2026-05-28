package structs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovieValueMethods(t *testing.T) {
	movie := &Movie{
		Title:          `Alien, "Director's Cut"`,
		ContentRating:  "R",
		Year:           1979,
		Genre:          "Sci-Fi",
		Library:        "Movies",
		MediaType:      "Movie",
		File:           "/media/alien.mkv",
		Hash:           "file-hash",
		Size:           1234,
		Duration:       7000,
		Container:      "mkv",
		Bitrate:        5000,
		VideoCodec:     "h264",
		Height:         1080,
		Width:          1920,
		Resolution:     "1080p",
		AudioCodec:     "aac",
		CriticRating:   8.5,
		AudienceRating: 9.1,
		MetadataHash:   "metadata-hash",
		QualityScore:   5.27,
	}

	if movie.GetTitle() != movie.Title {
		t.Fatalf("GetTitle() = %q, want %q", movie.GetTitle(), movie.Title)
	}
	if movie.GetYear() != movie.Year {
		t.Fatalf("GetYear() = %d, want %d", movie.GetYear(), movie.Year)
	}
	if movie.GetSizeBytes() != movie.Size {
		t.Fatalf("GetSizeBytes() = %d, want %d", movie.GetSizeBytes(), movie.Size)
	}
	if got, want := movie.GetUniqueTitle(), `Alien, "Director's Cut" 1979`; got != want {
		t.Fatalf("GetUniqueTitle() = %q, want %q", got, want)
	}
	if got, want := movie.CSVHeaders(), "title,rating,year,genre,library,media_type,file,hash,size,duration,container,bitrate,video_codec,height,width,resolution,audio_codec,critic_rating,audience_rating,metadata_hash,quality_score\n"; got != want {
		t.Fatalf("CSVHeaders() = %q, want %q", got, want)
	}
	if got, want := movie.ToCSV(), `"Alien, ""Director's Cut""",R,1979,Sci-Fi,Movies,Movie,/media/alien.mkv,file-hash,1234,7000,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,metadata-hash,5.27`+"\n"; got != want {
		t.Fatalf("ToCSV() = %q, want %q", got, want)
	}
}

func TestMovieCalculateQualityScore(t *testing.T) {
	movie := &Movie{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: "hevc", AudioCodec: "aac"}
	movie.CalculateQualityScore()
	if movie.QualityScore != 9.99 {
		t.Fatalf("QualityScore = %.2f, want 9.99", movie.QualityScore)
	}

	movie.Bitrate = 0
	movie.QualityScore = 99
	movie.CalculateQualityScore()
	if movie.QualityScore != 0 {
		t.Fatalf("QualityScore = %.2f with zero bitrate, want 0", movie.QualityScore)
	}
}

func TestEpisodeValueMethods(t *testing.T) {
	episode := &Episode{
		ShowTitle:      "The Expanse",
		SeasonNumber:   2,
		EpisodeNumber:  5,
		EpisodeTitle:   "Home",
		ContentRating:  "TV-14",
		Year:           2017,
		Library:        "TV",
		MediaType:      "TV Show",
		File:           "/media/expanse.mkv",
		Hash:           "file-hash",
		Size:           4321,
		Duration:       3000,
		Container:      "mkv",
		Bitrate:        4000,
		VideoCodec:     "h264",
		Height:         1080,
		Width:          1920,
		Resolution:     "1080p",
		AudioCodec:     "aac",
		CriticRating:   8.8,
		AudienceRating: 9.0,
		MetadataHash:   "metadata-hash",
		QualityScore:   4.12,
	}

	if episode.GetTitle() != "Home" {
		t.Fatalf("GetTitle() = %q, want %q", episode.GetTitle(), "Home")
	}
	if episode.GetYear() != 2017 {
		t.Fatalf("GetYear() = %d, want 2017", episode.GetYear())
	}
	if episode.GetSizeBytes() != 4321 {
		t.Fatalf("GetSizeBytes() = %d, want 4321", episode.GetSizeBytes())
	}
	if got, want := episode.GetUniqueTitle(), "The Expanse 2 Home 2017"; got != want {
		t.Fatalf("GetUniqueTitle() = %q, want %q", got, want)
	}
	if got, want := episode.GetShowUniqueTitle(), "The Expanse 2017"; got != want {
		t.Fatalf("GetShowUniqueTitle() = %q, want %q", got, want)
	}
	if got, want := episode.ToCSV(), "The Expanse,2,5,Home,TV-14,2017,TV,TV Show,/media/expanse.mkv,file-hash,4321,3000,mkv,4000,h264,1080,1920,1080p,aac,8.8,9.0,metadata-hash,4.12\n"; got != want {
		t.Fatalf("ToCSV() = %q, want %q", got, want)
	}

	episode.EpisodeTitle = `Home, "Again"`
	if got, want := episode.ToCSV(), `The Expanse,2,5,"Home, ""Again""",TV-14,2017,TV,TV Show,/media/expanse.mkv,file-hash,4321,3000,mkv,4000,h264,1080,1920,1080p,aac,8.8,9.0,metadata-hash,4.12`+"\n"; got != want {
		t.Fatalf("escaped ToCSV() = %q, want %q", got, want)
	}
}

func TestEpisodeCalculateQualityScore(t *testing.T) {
	episode := &Episode{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: "hevc", AudioCodec: "aac"}
	episode.CalculateQualityScore()
	if episode.QualityScore != 9.99 {
		t.Fatalf("QualityScore = %.2f, want 9.99", episode.QualityScore)
	}
}

func TestMovieCalculateQualityScoreBranches(t *testing.T) {
	codecs := []string{"av1", "hevc", "h265", "x265", "vp9", "h264", "x264", "vc1", "mpeg4", "divx", "xvid", "mpeg2video", "mpeg2", "unknown", "h264 10"}
	for _, codec := range codecs {
		movie := &Movie{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: codec, AudioCodec: "aac"}
		movie.CalculateQualityScore()
		if movie.QualityScore == 0 {
			t.Fatalf("QualityScore for codec %q = 0, want non-zero", codec)
		}
	}

	audioCodecs := []string{"dca", "dts", "dts-hd", "truehd", "flac", "ac3", "eac3", "aac", "mp3", "vorbis", "opus", "unknown"}
	for _, codec := range audioCodecs {
		movie := &Movie{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: "h264", AudioCodec: codec}
		movie.CalculateQualityScore()
		if movie.QualityScore == 0 {
			t.Fatalf("QualityScore for audio codec %q = 0, want non-zero", codec)
		}
	}

	movie := &Movie{Bitrate: 5000, VideoCodec: "h264", AudioCodec: "aac"}
	movie.CalculateQualityScore()
	if movie.QualityScore == 0 {
		t.Fatal("QualityScore with zero dimensions = 0, want non-zero")
	}
}

func TestEpisodeCalculateQualityScoreBranches(t *testing.T) {
	codecs := []string{"av1", "hevc", "h265", "x265", "vp9", "h264", "x264", "vc1", "mpeg4", "divx", "xvid", "mpeg2video", "mpeg2", "unknown", "h264 10"}
	for _, codec := range codecs {
		episode := &Episode{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: codec, AudioCodec: "aac"}
		episode.CalculateQualityScore()
		if episode.QualityScore == 0 {
			t.Fatalf("QualityScore for codec %q = 0, want non-zero", codec)
		}
	}

	audioCodecs := []string{"dca", "dts", "dts-hd", "truehd", "flac", "ac3", "eac3", "aac", "mp3", "vorbis", "opus", "unknown"}
	for _, codec := range audioCodecs {
		episode := &Episode{Bitrate: 5000, Width: 1920, Height: 1080, VideoCodec: "h264", AudioCodec: codec}
		episode.CalculateQualityScore()
		if episode.QualityScore == 0 {
			t.Fatalf("QualityScore for audio codec %q = 0, want non-zero", codec)
		}
	}

	episode := &Episode{Bitrate: 5000, VideoCodec: "h264", AudioCodec: "aac"}
	episode.CalculateQualityScore()
	if episode.QualityScore == 0 {
		t.Fatal("QualityScore with zero dimensions = 0, want non-zero")
	}

	episode.Bitrate = 0
	episode.QualityScore = 99
	episode.CalculateQualityScore()
	if episode.QualityScore != 0 {
		t.Fatalf("QualityScore with zero bitrate = %.2f, want 0", episode.QualityScore)
	}
}

func TestSongValueMethods(t *testing.T) {
	song := &Song{
		Title:        `Song, "Live"`,
		Year:         1999,
		Genre:        "Rock",
		AlbumTitle:   "The Album",
		ArtistName:   "The Artist",
		Library:      "Music",
		MediaType:    "Music",
		File:         "/media/song.flac",
		Hash:         "file-hash",
		Size:         1000,
		Duration:     200,
		Bitrate:      900,
		AudioCodec:   "flac",
		MetadataHash: "metadata-hash",
	}

	if song.GetTitle() != song.Title {
		t.Fatalf("GetTitle() = %q, want %q", song.GetTitle(), song.Title)
	}
	if song.GetYear() != 1999 {
		t.Fatalf("GetYear() = %d, want 1999", song.GetYear())
	}
	if song.GetSizeBytes() != 1000 {
		t.Fatalf("GetSizeBytes() = %d, want 1000", song.GetSizeBytes())
	}
	if got, want := song.GetUniqueTitle(), `Song, "Live" The Artist The Album 1999`; got != want {
		t.Fatalf("GetUniqueTitle() = %q, want %q", got, want)
	}
	if got, want := song.CSVHeaders(), "title,year,genre,album_title,artist_name,library,media_type,file,hash,size,duration,bitrate,audio_codec,metadata_hash\n"; got != want {
		t.Fatalf("CSVHeaders() = %q, want %q", got, want)
	}
	if got, want := song.ToCSV(), `"Song, ""Live""",1999,Rock,The Album,The Artist,Music,Music,/media/song.flac,file-hash,1000,200,900,flac,metadata-hash`+"\n"; got != want {
		t.Fatalf("ToCSV() = %q, want %q", got, want)
	}
}

func TestToMediaHelpers(t *testing.T) {
	movies := []*Movie{{Title: "Alien", Year: 1979}, {Title: "Aliens", Year: 1986}}

	media := ToMediaSlice(movies)
	if len(media) != 2 {
		t.Fatalf("len(media) = %d, want 2", len(media))
	}
	if media[0].GetUniqueTitle() != "Alien 1979" {
		t.Fatalf("media[0].GetUniqueTitle() = %q, want %q", media[0].GetUniqueTitle(), "Alien 1979")
	}

	item := ToMedia(movies[1])
	if item.GetUniqueTitle() != "Aliens 1986" {
		t.Fatalf("ToMedia().GetUniqueTitle() = %q, want %q", item.GetUniqueTitle(), "Aliens 1986")
	}
}

func TestGetMediaFromCSVFiles(t *testing.T) {
	dir := t.TempDir()

	movieCSV := filepath.Join(dir, "movies.csv")
	if err := os.WriteFile(movieCSV, []byte((&Movie{}).CSVHeaders()+"Alien,R,1979,Horror,Movies,Movie,/media/alien.mkv,file-hash,1234,7000,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,metadata-hash,5.27\n"), 0644); err != nil {
		t.Fatalf("WriteFile(movieCSV) error = %v", err)
	}
	movies, err := GetMoviesFromCSVFile(movieCSV)
	if err != nil {
		t.Fatalf("GetMoviesFromCSVFile() error = %v", err)
	}
	if len(movies) != 1 || movies[0].Title != "Alien" || movies[0].QualityScore != 5.27 {
		t.Fatalf("movies = %+v, want parsed movie", movies)
	}

	songCSV := filepath.Join(dir, "songs.csv")
	if err := os.WriteFile(songCSV, []byte((&Song{}).CSVHeaders()+"Song,1999,Rock,Album,Artist,Music,Music,/media/song.flac,file-hash,1000,200,900,flac,metadata-hash\n"), 0644); err != nil {
		t.Fatalf("WriteFile(songCSV) error = %v", err)
	}
	songs, err := GetSongsFromCSVFile(songCSV)
	if err != nil {
		t.Fatalf("GetSongsFromCSVFile() error = %v", err)
	}
	if len(songs) != 1 || songs[0].Title != "Song" || songs[0].ArtistName != "Artist" {
		t.Fatalf("songs = %+v, want parsed song", songs)
	}

	episodeCSV := filepath.Join(dir, "episodes.csv")
	if err := os.WriteFile(episodeCSV, []byte((&Episode{}).CSVHeaders()+"Show,1,2,Episode,TV-14,2020,TV,TV Show,/media/show.mkv,file-hash,2000,3000,mkv,4000,h264,720,1280,720p,aac,8.0,8.5,metadata-hash,4.25\n"), 0644); err != nil {
		t.Fatalf("WriteFile(episodeCSV) error = %v", err)
	}
	episodes, err := GetEpisodesFromCSVFile(episodeCSV)
	if err != nil {
		t.Fatalf("GetEpisodesFromCSVFile() error = %v", err)
	}
	if len(episodes) != 1 || episodes[0].ShowTitle != "Show" || episodes[0].EpisodeNumber != 2 {
		t.Fatalf("episodes = %+v, want parsed episode", episodes)
	}
}

func TestCSVReadersTreatBlankNumericFieldsAsZero(t *testing.T) {
	movieRow := []string{
		"Alien", "R", "", "Horror", "Movies", "Movie", "/media/alien.mkv",
		"file-hash", "", "", "mkv", "", "h264", "", "",
		"1080p", "aac", "", "", "metadata-hash", "",
	}
	movies, err := GetMoviesFromCSVFile(writeCSVFixture(t, (&Movie{}).CSVHeaders(), movieRow))
	if err != nil {
		t.Fatalf("GetMoviesFromCSVFile() error = %v", err)
	}
	if movies[0].Year != 0 || movies[0].Size != 0 || movies[0].QualityScore != 0 {
		t.Fatalf("movie blank numerics = %+v, want zero values", movies[0])
	}

	songRow := []string{
		"Song", "", "Rock", "Album", "Artist", "Music", "Music",
		"/media/song.flac", "file-hash", "", "", "", "flac", "metadata-hash",
	}
	songs, err := GetSongsFromCSVFile(writeCSVFixture(t, (&Song{}).CSVHeaders(), songRow))
	if err != nil {
		t.Fatalf("GetSongsFromCSVFile() error = %v", err)
	}
	if songs[0].Year != 0 || songs[0].Size != 0 || songs[0].Bitrate != 0 {
		t.Fatalf("song blank numerics = %+v, want zero values", songs[0])
	}

	episodeRow := []string{
		"Show", "", "", "Episode", "TV-14", "", "TV", "TV Show",
		"/media/show.mkv", "file-hash", "", "", "mkv", "",
		"h264", "", "", "720p", "aac", "", "", "metadata-hash", "",
	}
	episodes, err := GetEpisodesFromCSVFile(writeCSVFixture(t, (&Episode{}).CSVHeaders(), episodeRow))
	if err != nil {
		t.Fatalf("GetEpisodesFromCSVFile() error = %v", err)
	}
	if episodes[0].Year != 0 || episodes[0].Size != 0 || episodes[0].QualityScore != 0 {
		t.Fatalf("episode blank numerics = %+v, want zero values", episodes[0])
	}
}

func TestCSVReadersReturnOpenErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.csv")

	if _, err := GetMoviesFromCSVFile(missing); err == nil {
		t.Fatal("GetMoviesFromCSVFile() error = nil, want error")
	}
	if _, err := GetSongsFromCSVFile(missing); err == nil {
		t.Fatal("GetSongsFromCSVFile() error = nil, want error")
	}
	if _, err := GetEpisodesFromCSVFile(missing); err == nil {
		t.Fatal("GetEpisodesFromCSVFile() error = nil, want error")
	}
}

func TestCSVReadersReturnMalformedCSVErrors(t *testing.T) {
	cases := []struct {
		name string
		read func(string) error
	}{
		{name: "movie", read: func(path string) error {
			_, err := GetMoviesFromCSVFile(path)
			return err
		}},
		{name: "song", read: func(path string) error {
			_, err := GetSongsFromCSVFile(path)
			return err
		}},
		{name: "episode", read: func(path string) error {
			_, err := GetEpisodesFromCSVFile(path)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.csv")
			if err := os.WriteFile(path, []byte(`"unterminated`), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if err := tc.read(path); err == nil {
				t.Fatal("CSV reader error = nil, want malformed CSV error")
			}
		})
	}
}

func TestMovieCSVReaderReturnsParseErrors(t *testing.T) {
	base := []string{
		"Alien", "R", "1979", "Horror", "Movies", "Movie", "/media/alien.mkv",
		"file-hash", "1234", "7000", "mkv", "5000", "h264", "1080", "1920",
		"1080p", "aac", "8.5", "9.1", "metadata-hash", "5.27",
	}
	cases := []struct {
		name  string
		index int
		value string
	}{
		{name: "year", index: 2, value: "bad"},
		{name: "size", index: 8, value: "bad"},
		{name: "duration", index: 9, value: "bad"},
		{name: "bitrate", index: 11, value: "bad"},
		{name: "height", index: 13, value: "bad"},
		{name: "width", index: 14, value: "bad"},
		{name: "critic rating", index: 17, value: "bad"},
		{name: "audience rating", index: 18, value: "bad"},
		{name: "quality score", index: 20, value: "bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := append([]string(nil), base...)
			row[tc.index] = tc.value
			path := writeCSVFixture(t, (&Movie{}).CSVHeaders(), row)

			if _, err := GetMoviesFromCSVFile(path); err == nil {
				t.Fatal("GetMoviesFromCSVFile() error = nil, want error")
			}
		})
	}
}

func TestSongCSVReaderReturnsParseErrors(t *testing.T) {
	base := []string{
		"Song", "1999", "Rock", "Album", "Artist", "Music", "Music",
		"/media/song.flac", "file-hash", "1000", "200", "900", "flac", "metadata-hash",
	}
	cases := []struct {
		name  string
		index int
		value string
	}{
		{name: "year", index: 1, value: "bad"},
		{name: "size", index: 9, value: "bad"},
		{name: "duration", index: 10, value: "bad"},
		{name: "bitrate", index: 11, value: "bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := append([]string(nil), base...)
			row[tc.index] = tc.value
			path := writeCSVFixture(t, (&Song{}).CSVHeaders(), row)

			if _, err := GetSongsFromCSVFile(path); err == nil {
				t.Fatal("GetSongsFromCSVFile() error = nil, want error")
			}
		})
	}
}

func TestEpisodeCSVReaderReturnsParseErrors(t *testing.T) {
	base := []string{
		"Show", "1", "2", "Episode", "TV-14", "2020", "TV", "TV Show",
		"/media/show.mkv", "file-hash", "2000", "3000", "mkv", "4000",
		"h264", "720", "1280", "720p", "aac", "8.0", "8.5", "metadata-hash", "4.25",
	}
	cases := []struct {
		name  string
		index int
		value string
	}{
		{name: "season", index: 1, value: "bad"},
		{name: "episode", index: 2, value: "bad"},
		{name: "year", index: 5, value: "bad"},
		{name: "size", index: 10, value: "bad"},
		{name: "duration", index: 11, value: "bad"},
		{name: "bitrate", index: 13, value: "bad"},
		{name: "height", index: 15, value: "bad"},
		{name: "width", index: 16, value: "bad"},
		{name: "critic rating", index: 19, value: "bad"},
		{name: "audience rating", index: 20, value: "bad"},
		{name: "quality score", index: 22, value: "bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := append([]string(nil), base...)
			row[tc.index] = tc.value
			path := writeCSVFixture(t, (&Episode{}).CSVHeaders(), row)

			if _, err := GetEpisodesFromCSVFile(path); err == nil {
				t.Fatal("GetEpisodesFromCSVFile() error = nil, want error")
			}
		})
	}
}

func writeCSVFixture(t *testing.T, header string, row []string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fixture.csv")
	line := strings.Join(row, ",") + "\n"
	if err := os.WriteFile(path, []byte(header+line), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
