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

func AnalyzeComparisons(options AnalyzeComparisonsOptions) ComparisonResult {
	startTime := time.Now()

	result := ComparisonResult{
		SharedAnime: make(map[string][]SharedEntry),
		SharedManga: make(map[string][]SharedEntry),
	}

	// map of MediaId -> ComparatorEntry
	comparatorAnimeMap := buildMediaMap(options.ComparatorAnimeList)
	comparatorMangaMap := buildMediaMap(options.ComparatorMangaList)

	// Shared anime
	for _, creatorList := range options.CreatorAnimeList.Lists {
		for _, creatorEntry := range creatorList.Entries {
			if comparatorEntry, found := comparatorAnimeMap[creatorEntry.MediaId]; found {
				if creatorEntry.Status == comparatorEntry.Status {
					if creatorEntry.Score > 10 {
						creatorEntry.Score = creatorEntry.Score / 10
					}
					if comparatorEntry.Score > 10 {
						comparatorEntry.Score = comparatorEntry.Score / 10
					}

					sharedEntry := SharedEntry{
						MediaID:         creatorEntry.MediaId,
						Media:           creatorEntry.Media,
						CreatorScore:    creatorEntry.Score,
						ComparatorScore: comparatorEntry.Score,
					}
					result.SharedAnime[creatorList.Status] = append(result.SharedAnime[creatorList.Status], sharedEntry)
				}
			}
		}
	}

	// Shared manga
	for _, creatorList := range options.CreatorMangaList.Lists {
		for _, creatorEntry := range creatorList.Entries {
			if comparatorEntry, found := comparatorMangaMap[creatorEntry.MediaId]; found {
				if creatorEntry.Score > 10 {
					creatorEntry.Score = creatorEntry.Score / 10
				}
				if comparatorEntry.Score > 10 {
					comparatorEntry.Score = comparatorEntry.Score / 10
				}

				if creatorEntry.Status == comparatorEntry.Status {
					sharedEntry := SharedEntry{
						MediaID:         creatorEntry.MediaId,
						Media:           creatorEntry.Media,
						CreatorScore:    creatorEntry.Score,
						ComparatorScore: comparatorEntry.Score,
					}
					result.SharedManga[creatorList.Status] = append(result.SharedManga[creatorList.Status], sharedEntry)
				}
			}
		}
	}

	elapsedTime := time.Since(startTime)
	log.Printf("AnalyzeComparisons took %s", elapsedTime)

	return result
}

// map of MediaId -> MediaListEntry
func buildMediaMap(collection types.MediaListCollection) map[int]types.MediaListEntry {
	mediaMap := make(map[int]types.MediaListEntry)
	for _, list := range collection.Lists {
		for _, entry := range list.Entries {
			mediaMap[entry.MediaId] = entry
		}
	}
	return mediaMap
}
