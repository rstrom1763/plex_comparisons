package main

import plex "github.com/jrudio/go-plex-client"

type Episode struct {
	MetaDataObject plex.Metadata
}

func (e *Episode) getShowTitle() string {
	return e.MetaDataObject.GrandparentTitle
}

func (e *Episode) getTitle() string {
	return e.MetaDataObject.Title
}

func (e *Episode) getSeasonNumber() int {
	return int(e.MetaDataObject.ParentIndex)
}

func (e *Episode) getEpisodeNumber() int {
	return int(e.MetaDataObject.Index)
}
