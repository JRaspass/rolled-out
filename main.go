package main

import (
	"embed"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/JRaspass/rolled-out/assets"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db = sqlx.MustOpen("postgres", "")

//go:embed favicon.ico
var favicon embed.FS

func main() {
	// mixedCase boundary, e.g. fo(o)(B)ar or SQ(L)(B)ar.
	boundary := regexp.MustCompile(`([a-z])([A-Z])|([A-Z])([A-Z][a-z])`)
	db.MapperFunc(func(s string) string {
		return strings.ToLower(boundary.ReplaceAllString(s, "${1}${3}_${2}${4}"))
	})

	// Scrape every hour, on the hour.
	go func() {
		// Sleep until the top of the hour.
		time.Sleep(time.Until(time.Now().Truncate(time.Hour).Add(time.Hour)))

		tick := time.Tick(time.Hour)

		scrape(db)

		for range tick {
			scrape(db)
		}
	}()

	r := chi.NewRouter()

	r.Use(
		middleware.RealIP,
		middleware.Logger,
		errorMiddleware,
		middleware.Recoverer,
	)

	// A simple 404 handler that writes no body so that errorMiddleware can.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r.Get("/", ranks)
	r.Get("/about", about)
	r.Get("/dist/*", assets.Serve)
	r.Get("/favicon.ico", http.FileServer(http.FS(favicon)).ServeHTTP)
	r.Get("/grid", grid)
	r.Get("/latest", latest)
	r.Get("/latest/top-10", latest)
	r.Get("/latest/top-3", latest)
	r.Get("/players/{player}", player)
	r.Post("/players/{player}", playerAction)
	r.With(videosAuth).Get("/videos", videos)
	r.With(videosAuth).Post("/videos", videosAdd)
	r.With(videosAuth).Get("/videos/{id}/delete", videosDelete)
	r.With(videosAuth).Post("/videos/{id}/delete", videosDeleteConfirm)
	r.Get("/videos/{stage}", videosRuns)
	r.Get("/{world}", ranks)
	r.Get("/{world}/{stage}", ranks)

	log.Println("Listening…")
	panic(http.ListenAndServe(":8080", r))
}
