package main

import plex "github.com/jrudio/go-plex-client"

type Season struct {
	MetaDataObject plex.Metadata
	Episodes       map[int]Episode
}

func (s *Season) getShowTitle() string {
	return s.MetaDataObject.ParentTitle
}

func (s *Season) getShowYear() string {
	return s.MetaDataObject.OriginallyAvailableAt
}

func (s *Season) getSeasonNumber() int {
	return int(s.MetaDataObject.Index)
}

func (s *Season) addEpisode(episode Episode) {
	s.Episodes[int(episode.getEpisodeNumber())] = episode
}
