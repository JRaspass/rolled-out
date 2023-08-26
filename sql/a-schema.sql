CREATE TYPE goal AS ENUM ('Blue', 'Green', 'Red');

CREATE TABLE runs (
    stage_id       uuid      NOT NULL,
    date           timestamp NOT NULL,
    goal           goal      NOT NULL,
    player         text      NOT NULL,
    time_remaining bigint    NOT NULL,
    time_taken     bigint    NOT NULL,
    PRIMARY KEY (stage_id, goal, player)
);

CREATE TABLE videos (
    stage_id       uuid   NOT NULL,
    goal           goal   NOT NULL,
    player         text   NOT NULL,
    time_remaining bigint NOT NULL,
    video_author   text,
    video_title    text,
    video_url      text   NOT NULL UNIQUE,
    PRIMARY KEY (stage_id, goal, player)
);

CREATE MATERIALIZED VIEW points AS WITH window_funcs AS (
    SELECT *,
           min(time_remaining) OVER (PARTITION BY stage_id) min_time_remaining,
           max(time_remaining) OVER (PARTITION BY stage_id) max_time_remaining,
           rank() OVER (PARTITION BY stage_id ORDER BY time_remaining DESC),
           row_number() OVER (PARTITION BY player, stage_id ORDER BY time_remaining DESC) clear
      FROM runs
), points_parts AS (
    SELECT *,
           floor(
               (    time_remaining - min_time_remaining) /
               (max_time_remaining - min_time_remaining)::decimal * 750
           ) points_time,
           CASE WHEN rank =  1 THEN 200
                WHEN rank =  2 THEN 150
                WHEN rank =  3 THEN 100
                WHEN rank =  4 THEN  75
                WHEN rank =  5 THEN  60
                WHEN rank =  6 THEN  50
                WHEN rank =  7 THEN  40
                WHEN rank =  8 THEN  30
                WHEN rank =  9 THEN  20
                WHEN rank = 10 THEN  10
                               ELSE   0
           END points_rank
      FROM window_funcs
) SELECT *, points_time + points_rank + 50 points FROM points_parts;

CREATE ROLE "rolled-out" WITH LOGIN;

ALTER MATERIALIZED VIEW points OWNER TO "rolled-out";

GRANT SELECT, INSERT, DELETE ON TABLE runs, videos TO "rolled-out";
