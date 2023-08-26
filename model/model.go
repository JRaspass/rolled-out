package model

import (
	_ "embed"
	"encoding/json"
	"time"
)

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
	ID    string `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Timer int    `json:"timer"`
	World *World `json:"-"`
}

type Video struct {
	Goal, Player, VideoURL  string
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
}

func (s *Stage) Path() string { return "/" + s.World.Slug + "/" + s.Slug }
func (w *World) Path() string { return "/" + w.Slug }

func (s *Stage) Scan(id any) error {
	*s = *Stages[string(id.([]byte))]
	return nil
}
