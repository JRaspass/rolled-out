package video

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type Video struct {
	Author, Title *string // Optional
	URL           string
}

func Parse(u string) (*Video, error) {
	videoURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	if videoURL.Scheme != "https" {
		return nil, errors.New("scheme must be https")
	}

	var video Video

	switch videoURL.Host {
	case "www.youtube.com":
		// youtube.com → youtu.be for consistency.
		videoURL, err = url.Parse("https://youtu.be/" + videoURL.Query().Get("v"))
		if err != nil {
			return nil, err
		}

		fallthrough
	case "youtu.be":
		oembed := struct {
			AuthorName *string `json:"author_name"`
			Title      *string `json:"title"`
		}{}

		res, err := http.Get("https://www.youtube.com/oembed?url=" + videoURL.String())
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if err := json.NewDecoder(res.Body).Decode(&oembed); err != nil {
			return nil, err
		}

		video.Author = oembed.AuthorName
		video.Title = oembed.Title
	}

	video.URL = videoURL.String()

	return &video, nil
}
