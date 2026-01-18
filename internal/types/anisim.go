package types

import (
	"time"
)

type Comparison struct {
	CreatorUsername    string
	ComparatorUsername string
	Creator            User
	Comparator         User
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

type MediaListCollection struct {
	Lists []MediaList `json:"lists"`
}

type MediaList struct {
	Name         string           `json:"name"`
	Status       string           `json:"status"`
	Entries      []MediaListEntry `json:"entries"`
	IsCustomList bool             `json:"isCustomList"`
}

type MediaListEntry struct {
	ID              int       `json:"id"`
	MediaId         int       `json:"mediaId"`
	Media           Media     `json:"media"`
	Progress        int       `json:"progress"`
	Score           float64   `json:"score"`
	StartedAt       FuzzyDate `json:"startedAt"`
	Status          string    `json:"status"`
	ProgressVolumes int       `json:"progressVolumes"`
}

type Media struct {
	Title      Title      `json:"title"`
	CoverImage CoverImage `json:"coverImage"`
}

type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
}

type CoverImage struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
}

type FuzzyDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
}
