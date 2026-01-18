package analyzer

import (
	"log"
	"time"

	"github.com/will-x86/anisim/internal/types"
)

type AnalyzeComparisonsOptions struct {
	CreatorAnimeList    types.MediaListCollection
	ComparatorAnimeList types.MediaListCollection
	CreatorMangaList    types.MediaListCollection
	ComparatorMangaList types.MediaListCollection
}

/*
	type MediaListCollection struct {
		Lists []MediaList `json:"lists"`
	}

	type MediaList struct {
		Name         string           `json:"name"`
		Status       string           `json:"status"`
		Entries      []MediaListEntry `json:"entries"`
		IsCustomList bool             `json:"isCustomList"`
	}
*/
func AnalyzeComparisons(options AnalyzeComparisonsOptions) ComparisonResult {
	startTime := time.Now()

	result := ComparisonResult{
		SharedAnime: make(map[string][]SharedEntry),
		SharedManga: make(map[string][]SharedEntry),
	}

	// Build a map of MediaId -> ComparatorEntry for O(1) lookups
	comparatorAnimeMap := buildMediaMap(options.ComparatorAnimeList)
	comparatorMangaMap := buildMediaMap(options.ComparatorMangaList)

	// Find shared anime
	for _, creatorList := range options.CreatorAnimeList.Lists {
		for _, creatorEntry := range creatorList.Entries {
			if comparatorEntry, found := comparatorAnimeMap[creatorEntry.MediaId]; found {
				sharedEntry := SharedEntry{
					Media:           creatorEntry.Media,
					CreatorScore:    creatorEntry.Score,
					ComparatorScore: comparatorEntry.Score,
				}
				result.SharedAnime[creatorList.Status] = append(result.SharedAnime[creatorList.Status], sharedEntry)
			}
		}
	}

	// Find shared manga
	for _, creatorList := range options.CreatorMangaList.Lists {
		for _, creatorEntry := range creatorList.Entries {
			if comparatorEntry, found := comparatorMangaMap[creatorEntry.MediaId]; found {
				sharedEntry := SharedEntry{
					Media:           creatorEntry.Media,
					CreatorScore:    creatorEntry.Score,
					ComparatorScore: comparatorEntry.Score,
				}
				result.SharedManga[creatorList.Status] = append(result.SharedManga[creatorList.Status], sharedEntry)
			}
		}
	}

	elapsedTime := time.Since(startTime)
	log.Printf("AnalyzeComparisons took %s", elapsedTime)

	return result
}

// create map of MediaId -> MediaListEntry for fast lookups
func buildMediaMap(collection types.MediaListCollection) map[int]types.MediaListEntry {
	mediaMap := make(map[int]types.MediaListEntry)
	for _, list := range collection.Lists {
		for _, entry := range list.Entries {
			mediaMap[entry.MediaId] = entry
		}
	}
	return mediaMap
}
