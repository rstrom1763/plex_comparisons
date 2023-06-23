package main

import (
	plex "github.com/jrudio/go-plex-client"
)

type Show struct {
	MetaDataObject plex.Metadata
	Seasons        map[int]Season
}

func (s *Show) getTitle() string {
	return s.MetaDataObject.Title
}

func (s *Show) getYear() int {
	return s.MetaDataObject.Year
}

func (s *Show) addSeason(season Season) {
	s.Seasons[int(season.getSeasonNumber())] = season
}

func (s *Show) getSeason(index int) Season {
	return s.Seasons[index]
}

func (s *Show) addEpisode(episode Episode) {
	season := s.Seasons[episode.getSeasonNumber()]
	season.addEpisode(episode)
	s.Seasons[episode.getEpisodeNumber()] = season
}
