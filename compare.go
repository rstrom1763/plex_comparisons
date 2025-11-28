package main

import (
	"fmt"
	"log"
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

func getMediaItems(path1 string, path2 string, mediaType string) ([]Media, []Media, error) {

	switch mediaType {
	case "movie":
		mediaList1, err := getMoviesFromCSVFile(path1)
		if err != nil {
			return nil, nil, err
		}
		mediaList2, err := getMoviesFromCSVFile(path2)
		if err != nil {
			return nil, nil, err
		}
		return toMediaSlice(mediaList1), toMediaSlice(mediaList2), nil
	case "show":
		mediaList1, err := getEpisodesFromCSVFile(path1)
		if err != nil {
			return nil, nil, err
		}
		mediaList2, err := getEpisodesFromCSVFile(path2)
		if err != nil {
			return nil, nil, err
		}
		return toMediaSlice(mediaList1), toMediaSlice(mediaList2), nil
	case "song":
		mediaList1, err := getSongsFromCSVFile(path1)
		if err != nil {
			return nil, nil, err
		}
		mediaList2, err := getSongsFromCSVFile(path2)
		if err != nil {
			return nil, nil, err
		}
		return toMediaSlice(mediaList1), toMediaSlice(mediaList2), nil
	}

	return nil, nil, fmt.Errorf("unknown media type: %s", mediaType)
}

func compare(dumpFilePath1 string, dumpFilePath2 string, mediaType string) {
	mediaItems1, mediaItems2, err := getMediaItems(dumpFilePath1, dumpFilePath2, mediaType)
	if err != nil {
		log.Fatal(err)
	}
	dump1NoHave, dump2NoHave := compareDumps(mediaItems1, mediaItems2)

	err = writeCSV(addNoHaveToPath(dumpFilePath1), dump1NoHave)
	if err != nil && err.Error() != "input is empty" {
		log.Fatal(err)
	}
	err = writeCSV(addNoHaveToPath(dumpFilePath2), dump2NoHave)
	if err != nil && err.Error() != "input is empty" {
		log.Fatal(err)
	}

}
