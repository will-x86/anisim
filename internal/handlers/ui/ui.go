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
	comparisons, err := h.queries.GetAllComparisons(ctx)
	if err != nil {
		log.Printf("Error fetching comparisons: %v", err)
		http.Error(w, "Failed to load comparisons", http.StatusInternalServerError)
		return
	}

	component := pages.Home(comparisons)
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

	result := analyzer.AnalyzeComparisons(analyzer.AnalyzeComparisonsOptions{
		CreatorAnimeList:    creatorAnimeList,
		CreatorMangaList:    creatorMangaList,
		ComparatorAnimeList: comparatorAnimeList,
		ComparatorMangaList: comparatorMangaList,
	})

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
		CreatorMeanScore:       pgtype.Float8{Float64: creatorUser.Statistics.Manga.MeanScore, Valid: true},
		//Comparator
		ComparatorID:              pgtype.Int4{Int32: int32(comparatorUser.ID), Valid: true},
		ComparatorName:            pgtype.Text{String: comparatorUser.Name, Valid: true},
		ComparatorAvatarLarge:     pgtype.Text{String: comparatorUser.Avatar.Large, Valid: true},
		ComparatorAvatarMedium:    pgtype.Text{String: comparatorUser.Avatar.Medium, Valid: true},
		ComparatorEpisodesWatched: pgtype.Int4{Int32: int32(comparatorUser.Statistics.Anime.EpisodesWatched), Valid: true},
		ComparatorMinutesWatched:  pgtype.Int4{Int32: int32(comparatorUser.Statistics.Anime.MinutesWatched), Valid: true},
		ComparatorChaptersRead:    pgtype.Int4{Int32: int32(comparatorUser.Statistics.Manga.ChaptersRead), Valid: true},
		ComparatorMeanScore:       pgtype.Float8{Float64: comparatorUser.Statistics.Manga.MeanScore, Valid: true},
	})
	if err != nil {
		log.Printf("Error creating comparison: %v", err)
		http.Error(w, "Failed to create comparison", http.StatusInternalServerError)
		return
	}

	// Store shared anime entries
	for _, entry := range result.AllSharedAnime {
		_, err := h.queries.CreateSharedEntry(ctx, db.CreateSharedEntryParams{
			ComparisonID:      comparison.ID,
			MediaType:         "anime",
			MediaID:           int32(entry.MediaID),
			MediaTitleRomaji:  entry.Media.Title.Romaji,
			MediaTitleEnglish: pgtype.Text{String: entry.Media.Title.English, Valid: entry.Media.Title.English != ""},
			MediaCoverLarge:   pgtype.Text{String: entry.Media.CoverImage.Large, Valid: true},
			MediaCoverMedium:  pgtype.Text{String: entry.Media.CoverImage.Medium, Valid: true},
			CreatorStatus:     entry.CreatorStatus,
			ComparatorStatus:  entry.ComparatorStatus,
			CreatorScore:      pgtype.Float8{Float64: entry.CreatorScore, Valid: entry.CreatorScore > 0},
			ComparatorScore:   pgtype.Float8{Float64: entry.ComparatorScore, Valid: entry.ComparatorScore > 0},
		})
		if err != nil {
			log.Printf("Error creating shared anime entry: %v", err)
		}
	}

	// Store shared manga entries
	for _, entry := range result.AllSharedManga {
		_, err := h.queries.CreateSharedEntry(ctx, db.CreateSharedEntryParams{
			ComparisonID:      comparison.ID,
			MediaType:         "manga",
			MediaID:           int32(entry.MediaID),
			MediaTitleRomaji:  entry.Media.Title.Romaji,
			MediaTitleEnglish: pgtype.Text{String: entry.Media.Title.English, Valid: entry.Media.Title.English != ""},
			MediaCoverLarge:   pgtype.Text{String: entry.Media.CoverImage.Large, Valid: true},
			MediaCoverMedium:  pgtype.Text{String: entry.Media.CoverImage.Medium, Valid: true},
			CreatorStatus:     entry.CreatorStatus,
			ComparatorStatus:  entry.ComparatorStatus,
			CreatorScore:      pgtype.Float8{Float64: entry.CreatorScore, Valid: entry.CreatorScore > 0},
			ComparatorScore:   pgtype.Float8{Float64: entry.ComparatorScore, Valid: entry.ComparatorScore > 0},
		})
		if err != nil {
			log.Printf("Error creating shared manga entry: %v", err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/comparisons/%d", comparison.ID), http.StatusSeeOther)
}
