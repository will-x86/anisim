package types

import (
	"time"
)

type Comparison struct {
	CreatorUsername    string
	ComparatorUsername string
	CreatorUser        User
	ComparatorUser     User
	Created            time.Time
}

type User struct {
	ID         int
	Name       string
	Avatar     Avatar
	CreatedAt  int64
	Statistics UserStatistics
}

type Avatar struct {
	Large  string
	Medium string
}

type UserStatistics struct {
	Anime AnimeStatistics
	Manga MangaStatistics
}
type AnimeStatistics struct {
	EpisodesWatched int
	MinutesWatched  int
}

type MangaStatistics struct {
	ChaptersRead int
	MeanScore    float64
}
