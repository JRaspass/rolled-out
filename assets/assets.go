package assets

import (
	"embed"
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed assets.json
var assetsJSON []byte

var Paths = map[string]string{}

func init() {
	var esbuild struct {
		Outputs map[string]struct{ Inputs map[string]any }
	}

	if err := json.Unmarshal(assetsJSON, &esbuild); err != nil {
		panic(err)
	}

	for dist, output := range esbuild.Outputs {
		if len(output.Inputs) != 1 {
			panic(dist + " has too many inputs")
		}

		for input := range output.Inputs {
			Paths[input] = "/" + dist
		}
	}
}

//go:embed dist
var dist embed.FS
var fs = http.FileServer(http.FS(dist)).ServeHTTP

func Serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=31536000, public, immutable")
	fs(w, r)
}
