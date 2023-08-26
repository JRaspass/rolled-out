package views

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

func comma(i int) string {
	switch {
	case i >= 1e6:
		return fmt.Sprintf("%d,%03d,%03d", i/1e6, i%1e6/1e3, i%1e3)
	case i >= 1e3:
		return fmt.Sprintf("%d,%03d", i/1e3, i%1e3)
	default:
		return fmt.Sprint(i)
	}
}

// delta returns the millisecond delta between two time.Duration.
func delta(a, b time.Duration) int {
	return int((a - b).Milliseconds())
}

// timeMinSec renders a time.Duration in minutes and seconds.
func timeMinSec(d time.Duration) string {
	s := d / time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", s)
	}

	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

// TimeSec renders a time.Duration in seconds to three decimal places.
// Exported because it's used in a route :-(
func TimeSec(d time.Duration) string {
	ms := d.Milliseconds()
	return fmt.Sprintf("%d.%03d", ms/1000, ms%1000)
}

// date renders a fuzzy HTML <time> tag of a time.Time.
func date(t time.Time) template.HTML {
	const day = 24 * time.Hour

	var sb strings.Builder

	sb.WriteString(t.Format(
		"<time datetime=" + time.RFC3339Nano +
			` title="2 Jan 2006 15:04:05 UTC">`,
	))

	diff := time.Until(t)
	past := diff < 0
	if past {
		diff = -diff
	}

	if diff < 28*day {
		switch {
		case diff < 2*time.Minute:
			sb.WriteString("a min")
		case diff < time.Hour:
			fmt.Fprintf(&sb, "%d mins", diff/time.Minute)
		case diff < 2*time.Hour:
			sb.WriteString("an hour")
		case diff < day:
			fmt.Fprintf(&sb, "%d hours", diff/time.Hour)
		case diff < 2*day:
			sb.WriteString("a day")
		default:
			fmt.Fprintf(&sb, "%d days", diff/day)
		}

		if past {
			sb.WriteString(" ago")
		}
	} else {
		sb.WriteString(t.Format("2 Jan 2006"))
	}

	sb.WriteString("</time>")

	return template.HTML(sb.String())
}
