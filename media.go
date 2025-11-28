package main

type Media interface {
	ToCSV() string
	CSVHeaders() string
	GetUniqueTitle() string
}

func toMediaSlice[T Media](media []T) []Media {
	result := make([]Media, len(media))
	for i, _ := range media {
		result[i] = media[i]
	}
	return result
}

// ToMedia Adapts input to be of Media type
func ToMedia[T Media](mediaItem T) Media {
	return mediaItem
}
