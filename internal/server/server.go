package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/will-x86/anisim/internal/anilist"
	"github.com/will-x86/anisim/internal/handlers/ui"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	router *chi.Mux
	server *http.Server
	dbPool *pgxpool.Pool
}

func New() *Server {
	anilist.SetupClient("https://graphql.anilist.co")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:pass@localhost:5432/anisim?sslmode=disable"
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Database connection established")

	s := &Server{
		router: chi.NewRouter(),
		dbPool: dbPool,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)
}

func (s *Server) setupRoutes() {
	fs := http.FileServer(http.Dir("./static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fs))

	uiHandler := ui.NewHandler(s.dbPool)

	s.router.Get("/", uiHandler.HandleIndex)
	s.router.Get("/comparisons/{id}", uiHandler.HandleComparisonDetail)
	s.router.Post("/comparisons", uiHandler.HandleCreateAniSimComparison)
}

func (s *Server) Start(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.dbPool.Close()
	return s.server.Shutdown(ctx)
}
