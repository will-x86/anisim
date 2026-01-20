package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/will-x86/anisim/internal/analyzer"
	"github.com/will-x86/anisim/internal/anilist"
	"github.com/will-x86/anisim/internal/anilist/recommendation"
	"github.com/will-x86/anisim/internal/db"
	"github.com/will-x86/anisim/internal/types"
	"github.com/will-x86/anisim/templates/pages"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(dbPool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(dbPool),
	}
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	comparisons, err := h.queries.GetAllComparisonsOnePerCombo(ctx)
	if err != nil {
		log.Printf("Error fetching comparisons: %v", err)
		http.Error(w, "Failed to load comparisons", http.StatusInternalServerError)
		return
	}

	component := pages.Home(comparisons)
	templ.Handler(component).ServeHTTP(w, r)
}

func (h *Handler) HandleRecommendation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	dbComparison, err := h.queries.GetComparison(ctx, int32(id))
	if err != nil {
		log.Printf("Error fetching comparison: %v", err)
		http.Error(w, "Comparison not found", http.StatusNotFound)
		return
	}
	mediaEntries, err := h.queries.GetMediaEntriesWithCache(ctx, int32(id))
	if err != nil {
		log.Printf("Error fetching media entries with cache: %v", err)
		http.Error(w, "Failed to load comparison data", http.StatusInternalServerError)
		return
	}
	recommendationResult := recommendation.GetRecommendations(dbComparison, mediaEntries)

	// Limit anime and manga recommendations separately
	animeRecs := recommendationResult.Anime
	if len(animeRecs) > 10 {
		animeRecs = animeRecs[:10]
	}
	mangaRecs := recommendationResult.Manga
	if len(mangaRecs) > 10 {
		mangaRecs = mangaRecs[:10]
	}

	creatorUser := types.User{
		ID:   int(dbComparison.CreatorID.Int32),
		Name: dbComparison.CreatorName.String,
		Avatar: types.Avatar{
			Large:  dbComparison.CreatorAvatarLarge.String,
			Medium: dbComparison.CreatorAvatarMedium.String,
		},
	}

	comparatorUser := types.User{
		ID:   int(dbComparison.ComparatorID.Int32),
		Name: dbComparison.ComparatorName.String,
		Avatar: types.Avatar{
			Large:  dbComparison.ComparatorAvatarLarge.String,
			Medium: dbComparison.ComparatorAvatarMedium.String,
		},
	}

	comparison := types.Comparison{
		ID:                 int(dbComparison.ID),
		CreatorUsername:    dbComparison.CreatorUsername,
		ComparatorUsername: dbComparison.ComparatorUsername,
		Creator:            creatorUser,
		Comparator:         comparatorUser,
		Created:            dbComparison.ComparisonDate.Time,
	}

	component := pages.Recommendations(comparison, animeRecs, mangaRecs)
	templ.Handler(component).ServeHTTP(w, r)
}
func (h *Handler) HandleComparisonDetail(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	dBcomparison, err := h.queries.GetComparison(ctx, int32(id))
	if err != nil {
		log.Printf("Error fetching comparison: %v", err)
		http.Error(w, "Comparison not found", http.StatusNotFound)
		return
	}

	sharedEntries, err := h.queries.GetSharedEntriesByComparison(ctx, int32(id))
	if err != nil {
		log.Printf("Error fetching shared entries: %v", err)
		http.Error(w, "Failed to load comparison data", http.StatusInternalServerError)
		return
	}

	creatorUser := types.User{
		ID:   int(dBcomparison.CreatorID.Int32),
		Name: dBcomparison.CreatorName.String,
		Avatar: types.Avatar{
			Large:  dBcomparison.CreatorAvatarLarge.String,
			Medium: dBcomparison.CreatorAvatarMedium.String,
		},
		Statistics: types.UserStatistics{
			Anime: types.AnimeStatistics{
				EpisodesWatched: int(dBcomparison.CreatorEpisodesWatched.Int32),
				MinutesWatched:  int(dBcomparison.CreatorMinutesWatched.Int32),
			},
			Manga: types.MangaStatistics{
				ChaptersRead: int(dBcomparison.CreatorChaptersRead.Int32),
				MeanScore:    dBcomparison.CreatorMeanScore.Float64,
			},
		},
	}

	comparatorUser := types.User{
		ID:   int(dBcomparison.ComparatorID.Int32),
		Name: dBcomparison.ComparatorName.String,
		Avatar: types.Avatar{
			Large:  dBcomparison.ComparatorAvatarLarge.String,
			Medium: dBcomparison.ComparatorAvatarMedium.String,
		},
		Statistics: types.UserStatistics{
			Anime: types.AnimeStatistics{
				EpisodesWatched: int(dBcomparison.ComparatorEpisodesWatched.Int32),
				MinutesWatched:  int(dBcomparison.ComparatorMinutesWatched.Int32),
			},
			Manga: types.MangaStatistics{
				ChaptersRead: int(dBcomparison.ComparatorChaptersRead.Int32),
				MeanScore:    dBcomparison.ComparatorMeanScore.Float64,
			},
		},
	}

	result := analyzer.ComparisonResult{
		AllSharedAnime: []analyzer.SharedEntry{},
		AllSharedManga: []analyzer.SharedEntry{},
	}
	for _, entry := range sharedEntries {
		sharedEntry := analyzer.SharedEntry{
			MediaID: int(entry.MediaID),
			Media: types.Media{
				Title: types.Title{
					Romaji:  entry.MediaTitleRomaji,
					English: entry.MediaTitleEnglish.String,
				},
				CoverImage: types.CoverImage{
					Large:  entry.MediaCoverLarge.String,
					Medium: entry.MediaCoverMedium.String,
				},
			},
			CreatorScore:     entry.CreatorScore.Float64,
			ComparatorScore:  entry.ComparatorScore.Float64,
			CreatorStatus:    entry.CreatorStatus,
			ComparatorStatus: entry.ComparatorStatus,
		}

		if entry.MediaType == "anime" {
			result.AllSharedAnime = append(result.AllSharedAnime, sharedEntry)
		} else {
			result.AllSharedManga = append(result.AllSharedManga, sharedEntry)
		}
	}

	comparison := types.Comparison{
		ID:                 int(dBcomparison.ID),
		CreatorUsername:    dBcomparison.CreatorUsername,
		ComparatorUsername: dBcomparison.ComparatorUsername,
		Creator:            creatorUser,
		Comparator:         comparatorUser,
		Created:            dBcomparison.ComparisonDate.Time,
	}

	component := pages.ComparisonDetail(comparison, result)
	templ.Handler(component).ServeHTTP(w, r)
}

// Get info from user, usernames etc
// Grab lists & user info from anilist api
// Enqueue all media into caching queue
// Get same media from AnalyzeComparisons
// Store comparison in DB
// Batch insert shared entries
func (h *Handler) HandleCreateAniSimComparison(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	creatorUsername := r.FormValue("creator_username")
	comparatorUsername := r.FormValue("comparator_username")

	if creatorUsername == "" || comparatorUsername == "" {
		http.Error(w, "Both usernames are required", http.StatusBadRequest)
		return
	}

	// Fetch comparison data from AniList API
	creatorUser, creatorAnimeList, creatorMangaList,
		comparatorUser, comparatorAnimeList, comparatorMangaList, err := anilist.GetComparisonData(
		creatorUsername,
		comparatorUsername,
	)
	if err != nil {
		var rateLimitErr *anilist.RateLimitError
		if errors.As(err, &rateLimitErr) {
			http.Error(w, fmt.Sprintf("AniList API rate limit exceeded. Please try again in %d seconds.", rateLimitErr.RetryAfter), http.StatusTooManyRequests)
			return
		}
		log.Printf("Error fetching comparison data: %v", err)
		http.Error(w, "Failed to fetch comparison data from AniList", http.StatusInternalServerError)
		return
	}

	go enqueueMediaList(ctx, h.queries, creatorAnimeList, "ANIME")
	go enqueueMediaList(ctx, h.queries, creatorMangaList, "MANGA")
	go enqueueMediaList(ctx, h.queries, comparatorAnimeList, "ANIME")
	go enqueueMediaList(ctx, h.queries, comparatorMangaList, "MANGA")

	// Detect rating scales for mean score normalization
	creatorUses100Scale := detectUserScoreScale(creatorAnimeList, creatorMangaList)
	comparatorUses100Scale := detectUserScoreScale(comparatorAnimeList, comparatorMangaList)

	// Normalize mean scores
	creatorMeanScore := normalizeUserScore(creatorUser.Statistics.Manga.MeanScore, creatorUses100Scale)
	comparatorMeanScore := normalizeUserScore(comparatorUser.Statistics.Manga.MeanScore, comparatorUses100Scale)

	comparison, err := h.queries.CreateComparison(ctx, db.CreateComparisonParams{
		CreatorUsername:    creatorUsername,
		ComparatorUsername: comparatorUsername,
		//Creator
		CreatorID:              pgtype.Int4{Int32: int32(creatorUser.ID), Valid: true},
		CreatorName:            pgtype.Text{String: creatorUser.Name, Valid: true},
		CreatorAvatarLarge:     pgtype.Text{String: creatorUser.Avatar.Large, Valid: true},
		CreatorAvatarMedium:    pgtype.Text{String: creatorUser.Avatar.Medium, Valid: true},
		CreatorEpisodesWatched: pgtype.Int4{Int32: int32(creatorUser.Statistics.Anime.EpisodesWatched), Valid: true},
		CreatorMinutesWatched:  pgtype.Int4{Int32: int32(creatorUser.Statistics.Anime.MinutesWatched), Valid: true},
		CreatorChaptersRead:    pgtype.Int4{Int32: int32(creatorUser.Statistics.Manga.ChaptersRead), Valid: true},
		CreatorMeanScore:       pgtype.Float8{Float64: creatorMeanScore, Valid: true},
		//Comparator
		ComparatorID:              pgtype.Int4{Int32: int32(comparatorUser.ID), Valid: true},
		ComparatorName:            pgtype.Text{String: comparatorUser.Name, Valid: true},
		ComparatorAvatarLarge:     pgtype.Text{String: comparatorUser.Avatar.Large, Valid: true},
		ComparatorAvatarMedium:    pgtype.Text{String: comparatorUser.Avatar.Medium, Valid: true},
		ComparatorEpisodesWatched: pgtype.Int4{Int32: int32(comparatorUser.Statistics.Anime.EpisodesWatched), Valid: true},
		ComparatorMinutesWatched:  pgtype.Int4{Int32: int32(comparatorUser.Statistics.Anime.MinutesWatched), Valid: true},
		ComparatorChaptersRead:    pgtype.Int4{Int32: int32(comparatorUser.Statistics.Manga.ChaptersRead), Valid: true},
		ComparatorMeanScore:       pgtype.Float8{Float64: comparatorMeanScore, Valid: true},
	})
	if err != nil {
		log.Printf("Error creating comparison: %v", err)
		http.Error(w, "Failed to create comparison", http.StatusInternalServerError)
		return
	}

	// Store ALL media from both users (shared + unique to each user)
	allEntries := buildAllMediaEntries(
		comparison.ID,
		creatorAnimeList,
		creatorMangaList,
		comparatorAnimeList,
		comparatorMangaList,
	)

	if len(allEntries) > 0 {
		count, err := h.queries.BatchCreateMediaEntries(ctx, allEntries)
		if err != nil {
			log.Printf("Error batch creating media entries: %v", err)
		} else {
			log.Printf("Created %d media entries", count)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/comparisons/%d", comparison.ID), http.StatusSeeOther)
}

// detectUserScoreScale checks if a user has any score > 10, indicating they use 100-point scale
func detectUserScoreScale(animeList types.MediaListCollection, mangaList types.MediaListCollection) bool {
	uses100Scale := false

	// Check anime scores
	for _, list := range animeList.Lists {
		for _, entry := range list.Entries {
			if entry.Score > 10 {
				uses100Scale = true
				break
			}
		}
		if uses100Scale {
			break
		}
	}

	// Check manga scores if not already detected
	if !uses100Scale {
		for _, list := range mangaList.Lists {
			for _, entry := range list.Entries {
				if entry.Score > 10 {
					uses100Scale = true
					break
				}
			}
			if uses100Scale {
				break
			}
		}
	}

	return uses100Scale
}

// normalizeUserScore normalizes a score to 0-10 scale based on user's rating system
func normalizeUserScore(score float64, uses100Scale bool) float64 {
	if uses100Scale && score > 0 {
		return score / 10.0
	}
	return score
}

// buildAllMediaEntries creates media entries for ALL media from both users
func buildAllMediaEntries(
	comparisonID int32,
	creatorAnimeList types.MediaListCollection,
	creatorMangaList types.MediaListCollection,
	comparatorAnimeList types.MediaListCollection,
	comparatorMangaList types.MediaListCollection,
) []db.BatchCreateMediaEntriesParams {
	// Detect rating scales for each user
	creatorUses100Scale := detectUserScoreScale(creatorAnimeList, creatorMangaList)
	comparatorUses100Scale := detectUserScoreScale(comparatorAnimeList, comparatorMangaList)

	log.Printf("Creator uses 100-point scale: %v", creatorUses100Scale)
	log.Printf("Comparator uses 100-point scale: %v", comparatorUses100Scale)

	// Map media_id -> entry data
	mediaMap := make(map[int32]*mediaEntryData)

	// Process creator's anime
	for _, list := range creatorAnimeList.Lists {
		for _, entry := range list.Entries {
			id := int32(entry.MediaId)
			if _, exists := mediaMap[id]; !exists {
				mediaMap[id] = &mediaEntryData{
					MediaID:          id,
					MediaType:        "anime",
					Media:            entry.Media,
					InCreatorList:    false,
					InComparatorList: false,
				}
			}
			mediaMap[id].InCreatorList = true
			mediaMap[id].CreatorStatus = entry.Status
			mediaMap[id].CreatorScore = normalizeUserScore(entry.Score, creatorUses100Scale)
		}
	}

	// Process creator's manga
	for _, list := range creatorMangaList.Lists {
		for _, entry := range list.Entries {
			id := int32(entry.MediaId)
			if _, exists := mediaMap[id]; !exists {
				mediaMap[id] = &mediaEntryData{
					MediaID:          id,
					MediaType:        "manga",
					Media:            entry.Media,
					InCreatorList:    false,
					InComparatorList: false,
				}
			}
			mediaMap[id].InCreatorList = true
			mediaMap[id].CreatorStatus = entry.Status
			mediaMap[id].CreatorScore = normalizeUserScore(entry.Score, creatorUses100Scale)
		}
	}

	// Process comparator's anime
	for _, list := range comparatorAnimeList.Lists {
		for _, entry := range list.Entries {
			id := int32(entry.MediaId)
			if _, exists := mediaMap[id]; !exists {
				mediaMap[id] = &mediaEntryData{
					MediaID:          id,
					MediaType:        "anime",
					Media:            entry.Media,
					InCreatorList:    false,
					InComparatorList: false,
				}
			}
			mediaMap[id].InComparatorList = true
			mediaMap[id].ComparatorStatus = entry.Status
			mediaMap[id].ComparatorScore = normalizeUserScore(entry.Score, comparatorUses100Scale)
		}
	}

	// Process comparator's manga
	for _, list := range comparatorMangaList.Lists {
		for _, entry := range list.Entries {
			id := int32(entry.MediaId)
			if _, exists := mediaMap[id]; !exists {
				mediaMap[id] = &mediaEntryData{
					MediaID:          id,
					MediaType:        "manga",
					Media:            entry.Media,
					InCreatorList:    false,
					InComparatorList: false,
				}
			}
			mediaMap[id].InComparatorList = true
			mediaMap[id].ComparatorStatus = entry.Status
			mediaMap[id].ComparatorScore = normalizeUserScore(entry.Score, comparatorUses100Scale)
		}
	}

	// Convert to batch params
	allEntries := make([]db.BatchCreateMediaEntriesParams, 0, len(mediaMap))
	for _, data := range mediaMap {
		allEntries = append(allEntries, db.BatchCreateMediaEntriesParams{
			ComparisonID:      comparisonID,
			MediaType:         data.MediaType,
			MediaID:           data.MediaID,
			MediaTitleRomaji:  data.Media.Title.Romaji,
			MediaTitleEnglish: pgtype.Text{String: data.Media.Title.English, Valid: data.Media.Title.English != ""},
			MediaCoverLarge:   pgtype.Text{String: data.Media.CoverImage.Large, Valid: true},
			MediaCoverMedium:  pgtype.Text{String: data.Media.CoverImage.Medium, Valid: true},
			CreatorStatus:     data.CreatorStatus,
			ComparatorStatus:  data.ComparatorStatus,
			CreatorScore:      pgtype.Float8{Float64: data.CreatorScore, Valid: data.CreatorScore > 0},
			ComparatorScore:   pgtype.Float8{Float64: data.ComparatorScore, Valid: data.ComparatorScore > 0},
			InCreatorList:     data.InCreatorList,
			InComparatorList:  data.InComparatorList,
		})
	}

	return allEntries
}

type mediaEntryData struct {
	MediaID          int32
	MediaType        string
	Media            types.Media
	InCreatorList    bool
	InComparatorList bool
	CreatorStatus    string
	CreatorScore     float64
	ComparatorStatus string
	ComparatorScore  float64
}

func enqueueMediaList(ctx context.Context, queries *db.Queries, collection types.MediaListCollection, mediaType string) {
	// Collect all unique media IDs
	mediaIDs := make(map[int32]bool)
	for _, list := range collection.Lists {
		for _, entry := range list.Entries {
			mediaIDs[int32(entry.MediaId)] = true
		}
	}

	batchParams := make([]db.BatchEnqueueMediaForCachingParams, 0, len(mediaIDs))
	for mediaID := range mediaIDs {
		batchParams = append(batchParams, db.BatchEnqueueMediaForCachingParams{
			MediaID:   mediaID,
			MediaType: mediaType,
		})
	}

	if len(batchParams) > 0 {
		_, err := queries.BatchEnqueueMediaForCaching(ctx, batchParams)
		if err != nil {
			// Duplicate key errors are expected and fine - media already queued
			log.Printf("Enqueued %d %s media items (some may have been duplicates)", len(batchParams), mediaType)
		} else {
			log.Printf("Enqueued %d %s media items", len(batchParams), mediaType)
		}
	}
}
