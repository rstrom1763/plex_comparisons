package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstrom1763/plex_comparisons/structs"
)

func TestInitMediaMap(t *testing.T) {
	movies := []*structs.Movie{
		{Title: "Alien", Year: 1979},
		{Title: "Aliens", Year: 1986},
	}

	got := initMediaMap(movies)
	if len(got) != 2 {
		t.Fatalf("len(map) = %d, want 2", len(got))
	}
	if got["Alien 1979"] != movies[0] {
		t.Fatalf("map[Alien 1979] = %+v, want first movie", got["Alien 1979"])
	}
}

func TestFindNotIn(t *testing.T) {
	alien := &structs.Movie{Title: "Alien", Year: 1979}
	aliens := &structs.Movie{Title: "Aliens", Year: 1986}

	got := findNotIn([]*structs.Movie{alien, aliens}, initMediaMap([]*structs.Movie{alien}))
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != aliens {
		t.Fatalf("got[0] = %+v, want Aliens", got[0])
	}
}

func TestCompareDumps(t *testing.T) {
	alien := &structs.Movie{Title: "Alien", Year: 1979}
	aliens := &structs.Movie{Title: "Aliens", Year: 1986}
	prometheus := &structs.Movie{Title: "Prometheus", Year: 2012}

	dump1NoHave, dump2NoHave := compareDumps(
		[]*structs.Movie{alien, aliens},
		[]*structs.Movie{alien, prometheus},
	)

	if len(dump1NoHave) != 1 || dump1NoHave[0] != prometheus {
		t.Fatalf("dump1NoHave = %+v, want Prometheus", dump1NoHave)
	}
	if len(dump2NoHave) != 1 || dump2NoHave[0] != aliens {
		t.Fatalf("dump2NoHave = %+v, want Aliens", dump2NoHave)
	}
}

func TestGetMediaItemsFromCSVReturnsUnknownMediaTypeError(t *testing.T) {
	if _, err := getMediaItemsFromCSV("unused.csv", "book"); err == nil {
		t.Fatal("getMediaItemsFromCSV() error = nil, want error")
	}
}

func TestGetMediaItemsFromCSVPropagatesReaderErrors(t *testing.T) {
	for _, mediaType := range []string{"movie", "show", "song"} {
		t.Run(mediaType, func(t *testing.T) {
			if _, err := getMediaItemsFromCSV(filepath.Join(t.TempDir(), "missing.csv"), mediaType); err == nil {
				t.Fatal("getMediaItemsFromCSV() error = nil, want reader error")
			}
		})
	}
}

func TestGetMediaItemsFromCSV(t *testing.T) {
	dir := t.TempDir()

	moviePath := filepath.Join(dir, "movies.csv")
	if err := os.WriteFile(moviePath, []byte((&structs.Movie{}).CSVHeaders()+"Alien,R,1979,Horror,Movies,Movie,/media/alien.mkv,file-hash,1234,7000,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,metadata-hash,5.27\n"), 0644); err != nil {
		t.Fatalf("WriteFile(moviePath) error = %v", err)
	}
	movies, err := getMediaItemsFromCSV(moviePath, "movie")
	if err != nil {
		t.Fatalf("getMediaItemsFromCSV(movie) error = %v", err)
	}
	if len(movies) != 1 || movies[0].GetUniqueTitle() != "Alien 1979" {
		t.Fatalf("movies = %+v, want Alien media item", movies)
	}

	episodePath := filepath.Join(dir, "episodes.csv")
	if err := os.WriteFile(episodePath, []byte((&structs.Episode{}).CSVHeaders()+"Show,1,2,Episode,TV-14,2020,TV,TV Show,/media/show.mkv,file-hash,2000,3000,mkv,4000,h264,720,1280,720p,aac,8.0,8.5,metadata-hash,4.25\n"), 0644); err != nil {
		t.Fatalf("WriteFile(episodePath) error = %v", err)
	}
	episodes, err := getMediaItemsFromCSV(episodePath, "show")
	if err != nil {
		t.Fatalf("getMediaItemsFromCSV(show) error = %v", err)
	}
	if len(episodes) != 1 || episodes[0].GetUniqueTitle() != "Show 1 Episode 2020" {
		t.Fatalf("episodes = %+v, want Show media item", episodes)
	}

	songPath := filepath.Join(dir, "songs.csv")
	if err := os.WriteFile(songPath, []byte((&structs.Song{}).CSVHeaders()+"Song,1999,Rock,Album,Artist,Music,Music,/media/song.flac,file-hash,1000,200,900,flac,metadata-hash\n"), 0644); err != nil {
		t.Fatalf("WriteFile(songPath) error = %v", err)
	}
	songs, err := getMediaItemsFromCSV(songPath, "song")
	if err != nil {
		t.Fatalf("getMediaItemsFromCSV(song) error = %v", err)
	}
	if len(songs) != 1 || songs[0].GetUniqueTitle() != "Song Artist Album 1999" {
		t.Fatalf("songs = %+v, want Song media item", songs)
	}
}

func TestCompareWritesNoHaveCSVs(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.csv")
	secondPath := filepath.Join(dir, "second.csv")

	header := (&structs.Movie{}).CSVHeaders()
	first := header +
		"Alien,R,1979,Horror,Movies,Movie,/media/alien.mkv,alien-hash,1234,7000,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,alien-metadata,5.27\n" +
		"Aliens,R,1986,Action,Movies,Movie,/media/aliens.mkv,aliens-hash,1235,7001,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,aliens-metadata,5.27\n"
	second := header +
		"Alien,R,1979,Horror,Movies,Movie,/media/alien.mkv,alien-hash,1234,7000,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,alien-metadata,5.27\n" +
		"Prometheus,R,2012,Sci-Fi,Movies,Movie,/media/prometheus.mkv,prometheus-hash,1236,7002,mkv,5000,h264,1080,1920,1080p,aac,8.5,9.1,prometheus-metadata,5.27\n"
	if err := os.WriteFile(firstPath, []byte(first), 0644); err != nil {
		t.Fatalf("WriteFile(firstPath) error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(second), 0644); err != nil {
		t.Fatalf("WriteFile(secondPath) error = %v", err)
	}

	if err := compare(firstPath, secondPath, "movie"); err != nil {
		t.Fatalf("compare() error = %v", err)
	}

	firstNoHave, err := os.ReadFile(addNoHaveToPath(firstPath))
	if err != nil {
		t.Fatalf("ReadFile(first no-have) error = %v", err)
	}
	if !containsAll(string(firstNoHave), []string{"Prometheus", "2012"}) {
		t.Fatalf("first no-have CSV = %q, want Prometheus", string(firstNoHave))
	}

	secondNoHave, err := os.ReadFile(addNoHaveToPath(secondPath))
	if err != nil {
		t.Fatalf("ReadFile(second no-have) error = %v", err)
	}
	if !containsAll(string(secondNoHave), []string{"Aliens", "1986"}) {
		t.Fatalf("second no-have CSV = %q, want Aliens", string(secondNoHave))
	}
}

func TestWriteCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "movies.csv")
	movies := []*structs.Movie{{Title: "Alien", Year: 1979}}

	if err := writeCSV(path, movies); err != nil {
		t.Fatalf("writeCSV() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := (&structs.Movie{}).CSVHeaders() + "Alien,,1979,,,,,,0,0,,0,,0,0,,,0.0,0.0,,0.00\n"
	if string(got) != want {
		t.Fatalf("written CSV = %q, want %q", string(got), want)
	}
}

func TestWriteCSVSkipsEmptyInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")

	if err := writeCSV(path, []*structs.Movie{}); err != nil {
		t.Fatalf("writeCSV() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("os.Stat() error = %v, want not exist", err)
	}
}

func TestCompareReturnsErrors(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.csv")
	if err := os.WriteFile(validPath, []byte((&structs.Movie{}).CSVHeaders()+"Alien,,1979,,,,,,0,0,,0,,0,0,,,0.0,0.0,,0.00\n"), 0644); err != nil {
		t.Fatalf("WriteFile(validPath) error = %v", err)
	}

	if err := compare(filepath.Join(dir, "missing.csv"), validPath, "movie"); err == nil {
		t.Fatal("compare() error = nil for first CSV error, want error")
	}
	if err := compare(validPath, filepath.Join(dir, "missing.csv"), "movie"); err == nil {
		t.Fatal("compare() error = nil for second CSV error, want error")
	}
}

func containsAll(s string, substrings []string) bool {
	for _, substring := range substrings {
		if !strings.Contains(s, substring) {
			return false
		}
	}
	return true
}
