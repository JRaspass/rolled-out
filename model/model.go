package model

import (
	_ "embed"
	"encoding/json"
	"time"
)

type Link struct {
	Group, Name, Path string
	Prev, Next        *Link
	Stage             *Stage
	World             *World
}

// Contains the superset of fields every different table needs.
// TODO Probably needs splitting apart then there's a JSON API.
type Run struct {
	AvgRank, Goal, Player                               string
	Clear, Points, PointsRank, PointsTime, Rank, Stages int
	Date                                                time.Time
	Stage                                               *Stage `db:"stage_id"`
	TimeRemaining, TimeTaken                            time.Duration
	VideoURL                                            *string
}

type Stage struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Slug  string        `json:"slug"`
	Timer time.Duration `json:"timer"`
	World *World        `json:"-"`
}

type Video struct {
	Goal, Player, VideoURL  string
	ID                      int
	Rank                    *int
	Stage                   *Stage `db:"stage_id"`
	TimeRemaining           time.Duration
	VideoAuthor, VideoTitle *string
}

type World struct {
	Code   string   `json:"code"`
	Name   string   `json:"name"`
	Slug   string   `json:"slug"`
	Sort   int      `json:"-"`
	Stages []*Stage `json:"stages"`
}

//go:embed worlds.json
var worldsJSON []byte

var Stages = map[string]*Stage{}
var Worlds = []*World{}
var RankLinks = []*Link{}

func init() {
	if err := json.Unmarshal(worldsJSON, &Worlds); err != nil {
		panic(err)
	}

	for i, world := range Worlds {
		world.Sort = i

		for _, stage := range world.Stages {
			stage.World = world

			Stages[stage.ID] = stage
		}
	}

	// Overall + Worlds + Stages.
	RankLinks = make([]*Link, 1, 1+len(Worlds)+len(Stages))
	RankLinks[0] = &Link{Name: "Overall", Path: "/"}

	for _, world := range Worlds {
		RankLinks = append(RankLinks, &Link{
			Group: world.Name,
			Name:  "All Stages (" + world.Name + ")",
			Path:  world.Path(),
			World: world,
		})

		for _, stage := range world.Stages {
			RankLinks = append(RankLinks, &Link{
				Group: world.Name,
				Name:  stage.Name,
				Path:  stage.Path(),
				Stage: stage,
				World: world,
			})
		}
	}

	for i, link := range RankLinks {
		if i == 0 {
			link.Prev = RankLinks[len(RankLinks)-1]
		} else {
			link.Prev = RankLinks[i-1]
		}

		if i == len(RankLinks)-1 {
			link.Next = RankLinks[0]
		} else {
			link.Next = RankLinks[i+1]
		}
	}
}

func (s *Stage) Path() string { return "/" + s.World.Slug + "/" + s.Slug }
func (w *World) Path() string { return "/" + w.Slug }

func (s *Stage) Scan(id any) error {
	*s = *Stages[string(id.([]byte))]
	return nil
}
