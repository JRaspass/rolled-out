package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/JRaspass/rolled-out/model"
	"github.com/JRaspass/rolled-out/video"
	"github.com/JRaspass/rolled-out/views"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/maps"
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

func random(w http.ResponseWriter, r *http.Request) {
	url := model.RankLinks[rand.IntN(len(model.RankLinks))].Path
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func ranks(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Link          *model.Link
		Links         []*model.Link
		RecentRecords map[string][]*model.Run
		Rows          []model.Run
	}{Links: model.RankLinks}

	// Find the link (or return 404).
	if i := slices.IndexFunc(data.Links, func(link *model.Link) bool {
		return link.Path == r.URL.Path
	}); i >= 0 {
		data.Link = data.Links[i]
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var title, description string

	if stage := data.Link.Stage; stage != nil {
		title = stage.World.Name + " / " + stage.Name

		var err error
		if data.Rows, err = stage.Runs(db); err != nil {
			panic(err)
		}

		if len(data.Rows) > 0 {
			description = fmt.Sprintf("🥇 %s %s",
				views.TimeSec(data.Rows[0].TimeRemaining), data.Rows[0].Player)
		}
	} else if world := data.Link.World; world != nil {
		title = world.Name

		var err error
		if data.Rows, err = world.Runs(db); err != nil {
			panic(err)
		}
	} else {
		title = "Overall Ranks"

		var err error
		if data.RecentRecords, err = model.RecentRecords(db); err != nil {
			panic(err)
		}

		if data.Rows, err = model.OverallRuns(db); err != nil {
			panic(err)
		}
	}

	views.Render(w, r, "ranks.html", title, description, data)
}

func videos(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Stages []*model.Stage
		Videos []model.Video
	}{Stages: maps.Values(model.Stages)}

	// Sort by stage then world, looking at you, "Worm".
	slices.SortFunc(data.Stages, func(a, b *model.Stage) int {
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.World.Code, b.World.Code),
		)
	})

	var err error
	if data.Videos, err = model.Videos(db); err != nil {
		panic(err)
	}

	views.Render(w, r, "videos.html", "Videos", "", data)
}

// TODO Validation, error messages, etc.
func videosAdd(w http.ResponseWriter, r *http.Request) {
	vid, err := video.Parse(r.PostFormValue("url"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		vid.Author,
		vid.Title,
		vid.URL,
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

func videosDelete(w http.ResponseWriter, r *http.Request) {
	video, err := model.GetVideo(db, chi.URLParam(r, "id"))
	if err != nil {
		panic(err)
	}

	views.Render(w, r, "videos-delete.html", "", "", video)
}

func videosDeleteConfirm(w http.ResponseWriter, r *http.Request) {
	db.MustExec("DELETE FROM videos WHERE id = $1", chi.URLParam(r, "id"))

	http.Redirect(w, r, "/videos", http.StatusSeeOther)
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
