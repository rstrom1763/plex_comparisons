package main

import plex "github.com/jrudio/go-plex-client"

type Episode struct {
	MetaDataObject plex.Metadata
}

func (s *Episode) getShowTitle() string {
	return s.MetaDataObject.GrandparentTitle
}

func (s *Episode) getTitle() string {
	return s.MetaDataObject.Title
}

func (s *Episode) getSeasonNumber() int {
	return int(s.MetaDataObject.ParentIndex)
}

func (s *Episode) getEpisodeNumber() int {
	return int(s.MetaDataObject.Index)
}
