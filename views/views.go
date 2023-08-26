package views

import (
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/JRaspass/rolled-out/assets"
	"github.com/JRaspass/rolled-out/model"
)

//go:embed *.html
var views embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"asset":        func(s ...string) string { return assets.Paths[strings.Join(s, "")] },
	"comma":        comma,
	"date":         date,
	"delta":        delta,
	"inc":          func(i int) int { return i + 1 },
	"lower":        strings.ToLower,
	"path":         url.PathEscape,
	"time_min_sec": timeMinSec,
	"time_sec":     TimeSec,
}).ParseFS(views, "*"))

func Render(w http.ResponseWriter, r *http.Request, name, title, description string, data any) {
	stash := struct {
		Data                           any
		Description, Name, Path, Title string
		Worlds                         []*model.World
	}{
		Data:        data,
		Description: description,
		Name:        name,
		Path:        r.URL.Path,
		Title:       title,
		Worlds:      model.Worlds,
	}

	w.Header().Set("Content-Security-Policy",
		"base-uri 'none'; default-src 'self'; frame-ancestors 'none'; "+
			"img-src 'self' data:",
	)

	if err := tmpl.ExecuteTemplate(w, name, stash); err != nil {
		panic(err)
	}
}
