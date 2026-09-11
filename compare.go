package main

import (
	"fmt"

	utils "github.com/rstrom1763/goUtils"
	projectutils "github.com/rstrom1763/plex_comparisons/utils"

	. "github.com/rstrom1763/plex_comparisons/structs"
)

func findNotIn[T Media](items []T, otherDumpItemsMap map[string]T) []T {
	var notIn []T

	for _, item := range items {

		_, exists := otherDumpItemsMap[item.GetUniqueTitle()]

		if !exists {
			notIn = append(notIn, item)
		}
	}
	return notIn
}

func initMediaMap[T Media](mediaItems []T) map[string]T {
	itemsMap := make(map[string]T)
	for _, mediaItem := range mediaItems {
		itemsMap[mediaItem.GetUniqueTitle()] = mediaItem
	}

	return itemsMap
}

func compareDumps[T Media](dump1 []T, dump2 []T) ([]T, []T) {

	var dump1NoHave []T
	var dump2NoHave []T
	dump1Map := initMediaMap(dump1)
	dump2Map := initMediaMap(dump2)

	dump1NoHave = findNotIn(dump2, dump1Map)
	dump2NoHave = findNotIn(dump1, dump2Map)

	return dump1NoHave, dump2NoHave
}

func getMediaItemsFromCSV(path1 string, mediaType string) ([]Media, error) {

	switch mediaType {
	case "movie":
		mediaList, err := GetMoviesFromCSVFile(path1)
		if err != nil {
			return nil, err
		}
		return ToMediaSlice(mediaList), nil
	case "show":
		mediaList, err := GetEpisodesFromCSVFile(path1)
		if err != nil {
			return nil, err
		}
		return ToMediaSlice(mediaList), nil
	case "song":
		mediaList, err := GetSongsFromCSVFile(path1)
		if err != nil {
			return nil, err
		}
		return ToMediaSlice(mediaList), nil
	}

	return nil, fmt.Errorf("unknown media type: %s", mediaType)
}

func compare(dumpFilePath1 string, dumpFilePath2 string, mediaType string) error {
	mediaItems1, err := getMediaItemsFromCSV(dumpFilePath1, mediaType)
	if err != nil {
		return fmt.Errorf("could not get Media items from csv: %w", err)
	}

	mediaItems2, err := getMediaItemsFromCSV(dumpFilePath2, mediaType)
	if err != nil {
		return fmt.Errorf("could not get Media items from csv: %w", err)
	}

	dump1NoHave, dump2NoHave := compareDumps(mediaItems1, mediaItems2)

	err = writeCSV(projectutils.AddNoHaveToPath(dumpFilePath1), dump1NoHave)
	if err != nil && err.Error() != "input is empty" {
		return err
	}
	err = writeCSV(projectutils.AddNoHaveToPath(dumpFilePath2), dump2NoHave)
	if err != nil && err.Error() != "input is empty" {
		return err
	}

	return nil
}

func getByteSumFromDumpFile(dumpPath string, mediaType string) (int64, error) {
	var byteSum int64

	if !utils.FileExists(dumpPath) {
		return 0, fmt.Errorf("%s does not exist", dumpPath)
	}

	items, err := getMediaItemsFromCSV(dumpPath, mediaType)
	if err != nil {
		return 0, fmt.Errorf("could not get media items from csv: %s", err.Error())
	}

	for _, item := range items {
		byteSum += item.GetSizeBytes()
	}

	return byteSum, nil

}
