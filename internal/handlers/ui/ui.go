package ui

import (
	"context"
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
	comparison := types.Comparison{
		CreatorUsername:    dBcomparison.CreatorUsername,
		ComparatorUsername: dBcomparison.ComparatorUsername,
		Created:            dBcomparison.ComparisonDate.Time,
	}
	creatorAnilistUser, err := anilist.GetBasicUserInfo(dBcomparison.CreatorUsername)
	if err != nil {
		log.Printf("Error fetching creator Anilist user: %v", err)
		http.Error(w, "Failed to load creator user data", http.StatusInternalServerError)
		return
	}
	// Gets basic user info, get media list later
	comparatorAnilistUser, err := anilist.GetBasicUserInfo(dBcomparison.ComparatorUsername)
	if err != nil {
		log.Printf("Error fetching comparator Anilist user: %v", err)
		http.Error(w, "Failed to load comparator user data", http.StatusInternalServerError)
		return
	}
	comparison.Creator = creatorAnilistUser
	comparison.Comparator = comparatorAnilistUser
	creatorAnimeList, creatorMangaList, err := anilist.GetUsersMediaListCollection(creatorAnilistUser.ID)
	if err != nil {
		log.Printf("Error fetching creator media list: %v", err)
		http.Error(w, "Failed to load creator media list", http.StatusInternalServerError)
		return
	}
	comparatorAnimeList, comparatorMangaList, err := anilist.GetUsersMediaListCollection(comparatorAnilistUser.ID)
	if err != nil {
		log.Printf("Error fetching comparator media list: %v", err)
		http.Error(w, "Failed to load comparator media list", http.StatusInternalServerError)
		return
	}
	analyzerOptions := analyzer.AnalyzeComparisonsOptions{
		CreatorAnimeList:    creatorAnimeList,
		CreatorMangaList:    creatorMangaList,
		ComparatorAnimeList: comparatorAnimeList,
		ComparatorMangaList: comparatorMangaList,
	}
	result := analyzer.AnalyzeComparisons(analyzerOptions)

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

	_, err := h.queries.CreateComparison(ctx, db.CreateComparisonParams{
		CreatorUsername:    creatorUsername,
		ComparatorUsername: comparatorUsername,
	})
	if err != nil {
		log.Printf("Error creating comparison: %v", err)
		http.Error(w, "Failed to create comparison", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
