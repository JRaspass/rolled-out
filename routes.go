package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/JRaspass/rolled-out/model"
	"github.com/JRaspass/rolled-out/views"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var trAbstractionStage = model.Stages["b268ff63-3ada-4cfa-a40e-2d217430f5fd"]

func about(w http.ResponseWriter, r *http.Request) {
	runs, err := trAbstractionStage.Runs(db)
	if err != nil {
		panic(err)
	}

	views.Render(w, r, "about.html", "About", "", runs)
}

// errorMiddleware writes HTML/JSON bodies for 4xx/5xx status codes.
func errorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// Write an error body for 4xx & 5xx if we have yet to write a body.
		if code := ww.Status(); code >= 400 && ww.BytesWritten() == 0 {
			if ww.Header().Get("Content-Type") == "application/json" {
				ww.Write([]byte("null\n"))
			} else {
				text := fmt.Sprintf("%d %s", code, http.StatusText(code))
				views.Render(w, r, "error.html", text, "", text)
			}
		}
	})
}

func grid(w http.ResponseWriter, r *http.Request) {
	views.Render(w, r, "grid.html", "Grid", "", nil)
}

func latest(w http.ResponseWriter, r *http.Request) {
	var runs []*model.Run

	where := "true"
	switch r.URL.Path {
	case "/latest/top-10":
		where = "rank <= 10"
	case "/latest/top-3":
		where = "rank <= 3"
	}

	if err := db.Select(
		&runs,
		`  SELECT date, player, points, rank, stage_id, time_remaining,
		          video_url
		     FROM points
		LEFT JOIN videos USING (stage_id, goal, player, time_remaining)
		    WHERE `+where+`
		 ORDER BY date DESC
		    LIMIT 100`,
	); err != nil {
		panic(err)
	}

	views.Render(w, r, "latest.html", "Latest", "", runs)
}

func player(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Golds, Silvers, Bronzes        int
		HideAdditional, HideIncomplete bool
		Player                         string
		Runs                           map[string][]model.Run
	}{Runs: map[string][]model.Run{}}

	data.Player, _ = url.PathUnescape(chi.URLParam(r, "player"))

	c, _ := r.Cookie("hide_additional")
	data.HideAdditional = c != nil

	c, _ = r.Cookie("hide_incomplete")
	data.HideIncomplete = c != nil

	runs, err := model.PlayerRuns(db, data.Player)
	if err != nil {
		panic(err)
	}

	if len(runs) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, run := range runs {
		switch run.Rank {
		case 1:
			data.Golds++
		case 2:
			data.Silvers++
		case 3:
			data.Bronzes++
		}

		data.Runs[run.Stage.ID] = append(data.Runs[run.Stage.ID], run)
	}

	desc := fmt.Sprintf("🥇 %d • 🥈 %d • 🥉 %d", data.Golds, data.Silvers, data.Bronzes)
	views.Render(w, r, "player.html", data.Player, desc, data)
}

func playerAction(w http.ResponseWriter, r *http.Request) {
	var cookie *http.Cookie
	switch r.PostFormValue("action") {
	case "hide_additional":
		cookie = &http.Cookie{Name: "hide_additional"}
	case "show_additional":
		cookie = &http.Cookie{Name: "hide_additional", MaxAge: -1}
	case "hide_incomplete":
		cookie = &http.Cookie{Name: "hide_incomplete"}
	case "show_incomplete":
		cookie = &http.Cookie{Name: "hide_incomplete", MaxAge: -1}
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
}

func ranks(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Rows  []model.Run
		Stage *model.Stage
		World *model.World
	}{}

	// Find the world (or return 404) if we have a world param.
	if slug := chi.URLParam(r, "world"); slug != "" {
		for _, world := range model.Worlds {
			if slug == world.Slug {
				data.World = world
				break
			}
		}

		if data.World == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Find the stage (or return 404) if we have a stage param.
		if slug := chi.URLParam(r, "stage"); slug != "" {
			for _, stage := range data.World.Stages {
				if slug == stage.Slug {
					data.Stage = stage
					break
				}
			}

			if data.Stage == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
	}

	var title, description string

	if stage := data.Stage; stage != nil {
		title = data.World.Name + " / " + data.Stage.Name

		var err error
		if data.Rows, err = stage.Runs(db); err != nil {
			panic(err)
		}

		if len(data.Rows) > 0 {
			description = fmt.Sprintf("🥇 %s %s",
				views.TimeSec(data.Rows[0].TimeRemaining), data.Rows[0].Player)
		}
	} else if world := data.World; world != nil {
		title = world.Name

		var err error
		if data.Rows, err = world.Runs(db); err != nil {
			panic(err)
		}
	} else {
		title = "Overall Ranks"

		var err error
		if data.Rows, err = model.OverallRuns(db); err != nil {
			panic(err)
		}
	}

	views.Render(w, r, "ranks.html", title, description, data)
}

func videos(w http.ResponseWriter, r *http.Request) {
	videos, err := model.Videos(db)
	if err != nil {
		panic(err)
	}

	views.Render(w, r, "videos.html", "Videos", "", videos)
}

// TODO Validation, error messages, etc.
func videosAdd(w http.ResponseWriter, r *http.Request) {
	oembed := struct {
		AuthorName *string `json:"author_name"`
		Title      *string `json:"title"`
	}{}

	videoURL := r.PostFormValue("url")

	if strings.HasPrefix(videoURL, "https://youtu.be/") {
		res, err := http.Get("https://www.youtube.com/oembed?url=" + videoURL)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		if err := json.NewDecoder(res.Body).Decode(&oembed); err != nil {
			panic(err)
		}

		log.Println(oembed)
	}

	goalTimePlayer := strings.SplitN(r.PostFormValue("run"), "-", 3)

	db.MustExec(
		`INSERT INTO videos
		       (goal, player, stage_id, time_remaining, video_author, video_title, video_url)
		VALUES (  $1,     $2,       $3,             $4,           $5,          $6,        $7)`,
		goalTimePlayer[0],
		goalTimePlayer[2],
		r.PostFormValue("stage"),
		goalTimePlayer[1],
		oembed.AuthorName,
		oembed.Title,
		videoURL,
	)

	http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
}

// Very simple authentication for now.
func videosAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()

		if user == os.Getenv("VIDEO_USER") && pass == os.Getenv("VIDEO_PASS") {
			next.ServeHTTP(w, r)
		} else {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
}

// TODO Nothing about this is video specific, replace with a generic API.
func videosRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := model.Stages[chi.URLParam(r, "stage")].Runs(db)
	if err != nil {
		panic(err)
	}

	if err := json.NewEncoder(w).Encode(runs); err != nil {
		panic(err)
	}
}
