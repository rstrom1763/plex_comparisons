package main

import (
	plex "github.com/jrudio/go-plex-client"
)

type Movie struct {
	MetaDataObject plex.Metadata
}

func (m *Movie) getTitle() string {
	return m.MetaDataObject.Title
}

func (m *Movie) getYear() int {
	return m.MetaDataObject.Year
}

func (m *Movie) getMetadata() plex.Metadata {
	return m.MetaDataObject
}
