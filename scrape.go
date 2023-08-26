package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/JRaspass/rolled-out/model"
	"github.com/ericchiang/css"
	"github.com/jmoiron/sqlx"
	"golang.org/x/net/html"
)

func scrape(db *sqlx.DB) {
	start := time.Now()

	log.Println("Scraping…")

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered. Error:\n", r)
		}

		log.Println("Scrape done in", time.Since(start).Truncate(time.Second))
	}()

	tx := db.MustBegin()
	defer tx.Rollback()

	tx.MustExec("DELETE FROM runs")

	for _, stage := range model.Stages {
		req, err := http.NewRequest("GET",
			"https://scores.rolledoutgame.com/stages/"+stage.ID, nil)
		if err != nil {
			panic(err)
		}

		req.Header.Set("User-Agent", "Rolled Out Info - https://rolledout.info")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Println(res.Status, stage.ID, stage.Name)
			continue
		}

		doc, err := html.Parse(res.Body)
		if err != nil {
			panic(err)
		}

		for _, li := range queryAll(query(doc, "ul.leaderboard-list"), "li") {
			var run model.Run

			// Date
			var err error
			if run.Date, err = time.Parse(
				"2006-01-02 15:04:05 MST",
				attr(query(li, ".leaderboard-list__entry__created-at"), "title"),
			); err != nil {
				panic(err)
			}

			// Goal
			if query(li, ".leaderboard-list__entry__warp-distance--warp2") != nil {
				run.Goal = "Green"
			} else if query(li, ".leaderboard-list__entry__warp-distance--warp3") != nil {
				run.Goal = "Red"
			} else {
				run.Goal = "Blue"
			}

			// Player
			for _, div := range queryAll(li, "div") {
				if attr(div, "class") == "" {
					run.Player = text(div)
					break
				}
			}

			// TimeRemaining
			div := query(li, `[title*="ticks/sec"]`)
			if run.TimeRemaining, err = time.ParseDuration(text(div) + "s"); err != nil {
				panic(err)
			}

			// TimeTaken
			if run.TimeTaken, err = time.ParseDuration(
				strings.SplitAfter(attr(div, "title"), "s")[0],
			); err != nil {
				panic(err)
			}

			tx.MustExec(
				`INSERT INTO runs
				             (date, goal, player, stage_id, time_remaining, time_taken)
				      VALUES (  $1,   $2,     $3,       $4,             $5,         $6)`,
				run.Date,
				run.Goal,
				run.Player,
				stage.ID,
				run.TimeRemaining,
				run.TimeTaken,
			)
		}
	}

	tx.MustExec("REFRESH MATERIALIZED VIEW points")

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func query(n *html.Node, s string) *html.Node {
	if nodes := queryAll(n, s); len(nodes) > 0 {
		return nodes[0]
	}
	return nil
}

func queryAll(n *html.Node, s string) []*html.Node {
	return css.MustParse(s).Select(n)
}

// FIXME Assumes the first child is a text node.
func text(n *html.Node) string {
	return strings.TrimSpace(n.FirstChild.Data)
}
