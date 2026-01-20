package recommendation

import (
	"log"
	"math"
	"sort"

	"github.com/will-x86/anisim/internal/db"
)

const (
	MaxPopularity = 942779.0 // AOT
	MaxFavorites  = 91446.0  // One-Piece
)

type TagWithRank struct {
	Name string
	Rank int
}

type TagSource struct {
	MediaTitle string
	Score      float64
}

// extractTags parses tag_names and tag_ranks arrays from database to TagWithRank slice
func extractTags(tagNamesInterface, tagRanksInterface any) []TagWithRank {
	if tagNamesInterface == nil || tagRanksInterface == nil {
		return []TagWithRank{}
	}

	// Parse tag names
	var tagNames []string
	if names, ok := tagNamesInterface.([]any); ok {
		for _, name := range names {
			if nameStr, ok := name.(string); ok {
				tagNames = append(tagNames, nameStr)
			}
		}
	}

	var tagRanks []int
	if ranks, ok := tagRanksInterface.([]any); ok {
		for _, rank := range ranks {
			switch v := rank.(type) {
			case int:
				tagRanks = append(tagRanks, v)
			case int32:
				tagRanks = append(tagRanks, int(v))
			case int64:
				tagRanks = append(tagRanks, int(v))
			}
		}
	}

	// Combine into TagWithRank slice
	tags := []TagWithRank{}
	/*
		minLen := len(tagNames)
		if len(tagRanks) < minLen {
			minLen = len(tagRanks)
		}*/
	minLen := min(len(tagNames), len(tagRanks))
	for i := range minLen {
		tags = append(tags, TagWithRank{
			Name: tagNames[i],
			Rank: tagRanks[i],
		})
	}

	return tags
}

type Recommendation struct {
	MediaID           int32
	MediaTitleRomaji  string
	MediaTitleEnglish string
	MediaCoverLarge   string
	MediaCoverMedium  string
	MediaType         string
	Score             float64
	AlignmentScore    float64
	QualityScore      float64
	RelevanceScore    float64

	// Metadata for display
	ComparatorScore  float64
	ComparatorStatus string
	AverageScore     int32
	Popularity       int32
	Favourites       int32

	// Matching info
	MatchedGenres []string
	MatchedTags   []string
	GenreMatch    float64
	TagMatch      float64
	TagSources    map[string][]TagSource // Shows what media creator liked with each tag
}

// RecommendationResult holds separate anime and manga recommendations
type RecommendationResult struct {
	Anime []Recommendation
	Manga []Recommendation
}

// GetRecommendations generates separate personalized recommendations for anime and manga
// based on what the comparator has watched but the creator hasn't
func GetRecommendations(comparison db.Comparison, mediaEntries []db.GetMediaEntriesWithCacheRow) RecommendationResult {
	// Build preferences separately for anime and manga
	animeRecommendations := generateRecommendationsForType(mediaEntries, "anime")
	mangaRecommendations := generateRecommendationsForType(mediaEntries, "manga")

	log.Printf("Generated %d anime and %d manga recommendations", len(animeRecommendations), len(mangaRecommendations))

	return RecommendationResult{
		Anime: animeRecommendations,
		Manga: mangaRecommendations,
	}
}

// generateRecommendationsForType generates recommendations for a specific media type (anime or manga)
func generateRecommendationsForType(mediaEntries []db.GetMediaEntriesWithCacheRow, mediaType string) []Recommendation {
	var recommendations []Recommendation

	// Build creator's genre and tag preferences from shared media they liked (filtered by type)
	creatorGenreScores := buildGenrePreferencesForType(mediaEntries, mediaType)
	creatorTagScores, allTagSources := buildTagPreferencesForType(mediaEntries, mediaType)

	for _, row := range mediaEntries {
		entry := row.MediaEntry

		// Only process entries of the specified media type
		if entry.MediaType != mediaType {
			continue
		}

		// Only recommend media that:
		// - Comparator has but creator doesn't
		// - Comparator didn't drop or pause (already filtered in SQL)
		if !entry.InComparatorList || entry.InCreatorList {
			continue
		}

		if entry.ComparatorStatus == "DROPPED" || entry.ComparatorStatus == "PAUSED" {
			continue
		}

		// Scores are already normalized to 0-10 scale in database
		comparatorScore := entry.ComparatorScore.Float64

		if comparatorScore == 0 {
			continue
		}

		rec := Recommendation{
			MediaID:           entry.MediaID,
			MediaTitleRomaji:  entry.MediaTitleRomaji,
			MediaTitleEnglish: entry.MediaTitleEnglish.String,
			MediaCoverLarge:   entry.MediaCoverLarge.String,
			MediaCoverMedium:  entry.MediaCoverMedium.String,
			MediaType:         entry.MediaType,
			ComparatorScore:   comparatorScore,
			ComparatorStatus:  entry.ComparatorStatus,
			AverageScore:      row.AverageScore.Int32,
			Popularity:        row.Popularity.Int32,
			Favourites:        row.Favourites.Int32,
		}

		// Extract tags with ranks
		tags := extractTags(row.TagNames, row.TagRanks)

		rec.AlignmentScore = calculateAlignment(comparatorScore, row.AverageScore.Int32)
		rec.QualityScore = calculateQuality(row.AverageScore.Int32, row.Popularity.Int32, row.Favourites.Int32)

		// Calculate relevance with genre/tag matching
		matchedGenres, matchedTags, genreMatch, tagMatch, tagSources := getMatchingGenresAndTags(row.Genres, tags, creatorGenreScores, creatorTagScores, allTagSources)
		rec.MatchedGenres = matchedGenres
		rec.MatchedTags = matchedTags
		rec.GenreMatch = genreMatch
		rec.TagMatch = tagMatch
		rec.TagSources = tagSources
		rec.RelevanceScore = calculateRelevanceWithGenresAndTags(comparatorScore, genreMatch, tagMatch)

		rec.Score = (rec.AlignmentScore * 0.4) + (rec.QualityScore * 0.3) + (rec.RelevanceScore * 0.3)

		// Boost if comparator completed it
		if entry.ComparatorStatus == "COMPLETED" {
			rec.Score *= 1.2
		}

		recommendations = append(recommendations, rec)
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// buildGenrePreferencesForType analyzes what genres the creator likes based on their scores for a specific media type
func buildGenrePreferencesForType(entries []db.GetMediaEntriesWithCacheRow, mediaType string) map[string]float64 {
	genreScores := make(map[string]float64)
	genreCounts := make(map[string]int)

	for _, row := range entries {
		entry := row.MediaEntry

		// Only media of the specified type that creator has watched and scored
		if entry.MediaType != mediaType || !entry.InCreatorList || !entry.CreatorScore.Valid {
			continue
		}

		creatorScore := entry.CreatorScore.Float64

		if creatorScore == 0 {
			continue
		}

		for _, genre := range row.Genres {
			genreScores[genre] += creatorScore
			genreCounts[genre]++
		}
	}

	genreAvgs := make(map[string]float64)
	for genre, total := range genreScores {
		genreAvgs[genre] = total / float64(genreCounts[genre])
	}

	return genreAvgs
}

// buildGenrePreferences analyzes what genres the creator likes based on their scores (legacy, keeps all types)
func buildGenrePreferences(entries []db.GetMediaEntriesWithCacheRow) map[string]float64 {
	genreScores := make(map[string]float64)
	genreCounts := make(map[string]int)

	for _, row := range entries {
		entry := row.MediaEntry

		// Only media creator has watched and scored
		if !entry.InCreatorList || !entry.CreatorScore.Valid {
			continue
		}

		creatorScore := entry.CreatorScore.Float64

		if creatorScore == 0 {
			continue
		}

		for _, genre := range row.Genres {
			genreScores[genre] += creatorScore
			genreCounts[genre]++
		}
	}

	genreAvgs := make(map[string]float64)
	for genre, total := range genreScores {
		genreAvgs[genre] = total / float64(genreCounts[genre])
	}

	return genreAvgs
}

// buildTagPreferencesForType analyzes what tags the creator likes based on their scores for a specific media type
// Returns both the average scores and the source media for each tag
func buildTagPreferencesForType(entries []db.GetMediaEntriesWithCacheRow, mediaType string) (map[string]float64, map[string][]TagSource) {
	tagScores := make(map[string]float64)
	tagCounts := make(map[string]int)
	tagSources := make(map[string][]TagSource)

	for _, row := range entries {
		entry := row.MediaEntry

		// Only media of the specified type that creator has watched and scored
		if entry.MediaType != mediaType || !entry.InCreatorList || !entry.CreatorScore.Valid {
			continue
		}

		creatorScore := entry.CreatorScore.Float64

		if creatorScore == 0 {
			continue
		}

		// Get media title
		mediaTitle := entry.MediaTitleEnglish.String
		if mediaTitle == "" {
			mediaTitle = entry.MediaTitleRomaji
		}

		// Extract tags with ranks
		tags := extractTags(row.TagNames, row.TagRanks)

		for _, tag := range tags {
			// Weight the tag by its rank (rank is 0-100, higher = more relevant)
			weight := float64(tag.Rank) / 100.0
			weightedScore := creatorScore * weight

			tagScores[tag.Name] += weightedScore
			tagCounts[tag.Name]++
			tagSources[tag.Name] = append(tagSources[tag.Name], TagSource{
				MediaTitle: mediaTitle,
				Score:      creatorScore,
			})
		}
	}

	tagAvgs := make(map[string]float64)
	for tag, total := range tagScores {
		tagAvgs[tag] = total / float64(tagCounts[tag])
	}

	return tagAvgs, tagSources
}

// buildTagPreferences analyzes what tags the creator likes based on their scores (legacy, keeps all types)
// Returns both the average scores and the source media for each tag
func buildTagPreferences(entries []db.GetMediaEntriesWithCacheRow) (map[string]float64, map[string][]TagSource) {
	tagScores := make(map[string]float64)
	tagCounts := make(map[string]int)
	tagSources := make(map[string][]TagSource)

	for _, row := range entries {
		entry := row.MediaEntry

		if !entry.InCreatorList || !entry.CreatorScore.Valid {
			continue
		}

		creatorScore := entry.CreatorScore.Float64

		if creatorScore == 0 {
			continue
		}

		// Get media title
		mediaTitle := entry.MediaTitleEnglish.String
		if mediaTitle == "" {
			mediaTitle = entry.MediaTitleRomaji
		}

		// Extract tags with ranks
		tags := extractTags(row.TagNames, row.TagRanks)

		for _, tag := range tags {
			// Weight the tag by its rank (rank is 0-100, higher = more relevant)
			weight := float64(tag.Rank) / 100.0
			weightedScore := creatorScore * weight

			tagScores[tag.Name] += weightedScore
			tagCounts[tag.Name]++
			tagSources[tag.Name] = append(tagSources[tag.Name], TagSource{
				MediaTitle: mediaTitle,
				Score:      creatorScore,
			})
		}
	}

	tagAvgs := make(map[string]float64)
	for tag, total := range tagScores {
		tagAvgs[tag] = total / float64(tagCounts[tag])
	}

	return tagAvgs, tagSources
}

// calculateAlignment measures how well this aligns with creator's tastes
// Higher comparator score relative to mean suggests creator would like it
func calculateAlignment(comparatorScore float64, averageScore int32) float64 {
	// If no average score, just use comparator's normalized score
	if averageScore == 0 {
		return comparatorScore / 10.0
	}

	meanScore := float64(averageScore) / 10.0

	// How much the comparator liked it relative to the mean
	// Normalize to 0-1 range
	alignment := (comparatorScore - meanScore) / 10.0

	// Clamp to 0-1
	if alignment < 0 {
		alignment = 0
	}
	if alignment > 1 {
		alignment = 1
	}

	return alignment
}

// calculateQuality measures the overall quality/popularity of the media
func calculateQuality(meanScore int32, popularity int32, favourites int32) float64 {
	scoreComponent := (float64(meanScore) / 100.0) * 0.9 // Actual quality is 90%

	popularityPercentile := math.Min(float64(popularity)/MaxPopularity, 1.0) * 0.05 // Only 5%
	favoritesPercentile := math.Min(float64(favourites)/MaxFavorites, 1.0) * 0.05   // Only 5%

	return scoreComponent + popularityPercentile + favoritesPercentile
}

// tagWithScore holds a tag and its relevance score for sorting
type tagWithScore struct {
	tag   string
	score float64
}

// getMatchingGenresAndTags finds which genres/tags match creator's preferences
func getMatchingGenresAndTags(mediaGenres []string, mediaTags []TagWithRank, creatorGenrePrefs map[string]float64, creatorTagPrefs map[string]float64, allTagSources map[string][]TagSource) ([]string, []string, float64, float64, map[string][]TagSource) {
	matchedGenres := []string{}
	genreScoreSum := 0.0

	for _, genre := range mediaGenres {
		if avgScore, exists := creatorGenrePrefs[genre]; exists {
			matchedGenres = append(matchedGenres, genre)
			genreScoreSum += avgScore / 10.0
		}
	}

	genreMatch := 0.0 // No matches = no contribution
	if len(matchedGenres) > 0 {
		genreMatch = genreScoreSum / float64(len(matchedGenres))
	}

	// Collect tags with their scores for sorting, weighted by rank
	tagScores := []tagWithScore{}
	tagScoreSum := 0.0
	totalWeight := 0.0

	for _, tag := range mediaTags {
		if avgScore, exists := creatorTagPrefs[tag.Name]; exists {
			// Weight by tag rank (rank is 0-100)
			weight := float64(tag.Rank) / 100.0

			tagScores = append(tagScores, tagWithScore{tag: tag.Name, score: avgScore}) // Store unweighted for sorting
			tagScoreSum += avgScore / 10.0 * weight                                     // Use weighted score for match calculation
			totalWeight += weight
		}
	}

	// Sort tags by creator preference score (highest first) and take top 5
	sort.Slice(tagScores, func(i, j int) bool {
		return tagScores[i].score > tagScores[j].score
	})

	matchedTags := []string{}
	filteredTagSources := make(map[string][]TagSource)
	maxTags := 5
	for i := 0; i < len(tagScores) && i < maxTags; i++ {
		tag := tagScores[i].tag
		matchedTags = append(matchedTags, tag)
		// Copy the sources for this tag
		if sources, exists := allTagSources[tag]; exists {
			filteredTagSources[tag] = sources
		}
	}

	tagMatch := 0.0 // No matches = no contribution
	if totalWeight > 0 {
		// Normalize by total weight instead of count to account for rank weighting
		tagMatch = tagScoreSum / totalWeight
	}

	return matchedGenres, matchedTags, genreMatch, tagMatch, filteredTagSources
}

func calculateRelevanceWithGenresAndTags(comparatorScore float64, genreMatch float64, tagMatch float64) float64 {
	// comparatorScore is already normalized to 0-10
	baseRelevance := comparatorScore / 10.0
	if baseRelevance > 1 {
		baseRelevance = 1
	}

	// Combine: 50% comparator score, 25% genre matching, 25% tag matching
	return (baseRelevance * 0.5) + (genreMatch * 0.25) + (tagMatch * 0.25)
}
