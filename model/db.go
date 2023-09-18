package model

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func OverallRuns(db *sqlx.DB) (runs []Run, err error) {
	return aggregateRuns(db, nil)
}

func PlayerRuns(db *sqlx.DB, player string) (runs []Run, err error) {
	err = db.Select(
		&runs,
		`  SELECT clear, date, goal, points, rank, stage_id, time_remaining,
		          video_url
		     FROM points
		LEFT JOIN videos USING (stage_id, goal, player, time_remaining)
		    WHERE player = $1
		 ORDER BY stage_id, goal`,
		player,
	)
	return
}

func RecentRecords(db *sqlx.DB) (map[string][]*Run, error) {
	var runs []*Run

	// Keep in sync with the view.
	err := db.Select(
		&runs,
		`WITH points_with_count AS (
		    SELECT *, COUNT(*) OVER (PARTITION BY player) record_count
		      FROM points
		     WHERE date >= now() - interval '3 days'
		       AND rank <= 3
		) SELECT date, player, rank, stage_id, time_remaining
		    FROM points_with_count
		   WHERE record_count >= 3
		ORDER BY date DESC`,
	)

	records := map[string][]*Run{}
	for _, run := range runs {
		records[run.Player] = append(records[run.Player], run)
	}

	return records, err
}

func GetVideo(db *sqlx.DB, id string) (*Video, error) {
	var video Video
	err := db.Get(
		&video,
		`  SELECT goal, id, player, rank, stage_id, time_remaining,
		          video_author, video_title, video_url
		     FROM videos
		LEFT JOIN points USING (stage_id, goal, player, time_remaining)
		    WHERE id = $1`,
		id,
	)
	return &video, err
}

func Videos(db *sqlx.DB) (videos []Video, err error) {
	err = db.Select(
		&videos,
		`  SELECT goal, id, player, rank, stage_id, time_remaining,
		          video_author, video_title, video_url
		     FROM videos
		LEFT JOIN points USING (stage_id, goal, player, time_remaining)
		 ORDER BY id DESC`,
	)
	return
}

func (s *Stage) Runs(db *sqlx.DB) ([]Run, error) {
	runs := []Run{} // Ensure we don't return a nil slice.
	err := db.Select(
		&runs,
		`  SELECT clear, date, goal, player, points, points_rank, points_time,
		          rank, time_remaining, video_url
		     FROM points
		LEFT JOIN videos USING (stage_id, goal, player, time_remaining)
		    WHERE stage_id = $1
		 ORDER BY rank, date, player`,
		s.ID,
	)
	return runs, err
}

func (w *World) Runs(db *sqlx.DB) (runs []Run, err error) {
	stageIDs := make([]string, len(w.Stages))
	for i, s := range w.Stages {
		stageIDs[i] = s.ID
	}

	return aggregateRuns(db, stageIDs)
}

func aggregateRuns(db *sqlx.DB, stageIDs []string) (runs []Run, err error) {
	var args []any
	var where string
	if len(stageIDs) > 0 {
		args = []any{pq.Array(stageIDs)}
		where = "AND stage_id = ANY($1)"
	}

	err = db.Select(
		&runs,
		` SELECT rank() OVER (ORDER BY sum(points) DESC) rank,
		         round(avg(rank), 2) avg_rank,
		         player, sum(points) points, count(*) stages,
		         sum(time_taken) time_taken
		    FROM points
		   WHERE clear = 1 `+where+`
		GROUP BY player
		ORDER BY rank, player`,
		args...,
	)
	return
}
