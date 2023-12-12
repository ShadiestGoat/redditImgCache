package main

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"time"

	"github.com/shadiestgoat/log"
	"github.com/shadiestgoat/redditImgCache/db"
)

type ImageRow struct {
	PostID    string `json:"-"`
	Image     string `json:"img"`
	IsNSFW    bool   `json:"nsfw"`
	Subreddit string `json:"-"`
	TrueSub   string `json:"-"`
	CreatedAt int    `json:"-"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type RedditImage interface {
	URL() string
	Width() int
	Height() int
}

type ImagePreview struct {
	Source struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"source"`
}

func (img ImagePreview) URL() string {
	return img.Source.URL
}
func (img ImagePreview) Width() int {
	return img.Source.Width
}
func (img ImagePreview) Height() int {
	return img.Source.Height
}

type ImageGallery struct {
	Source struct {
		URL    string `json:"u"`
		Width  int    `json:"x"`
		Height int    `json:"y"`
	} `json:"s"`
}

func (img ImageGallery) URL() string {
	return img.Source.URL
}
func (img ImageGallery) Width() int {
	return img.Source.Width
}
func (img ImageGallery) Height() int {
	return img.Source.Height
}

type Resp struct {
	Data struct {
		After  *string `json:"after"`
		Before *string `json:"before"`
		Total  int     `json:"dist"`

		Children []struct {
			Kind string `json:"kind"`
			Data struct {
				Name      string  `json:"name"`
				Thumbnail string  `json:"thumbnail"`
				CreatedAt float64 `json:"created_utc"`
				Preview   struct {
					Images []ImagePreview `json:"images"`
				} `json:"preview"`
				MediaMetadata map[string]ImageGallery `json:"media_metadata"`
				Over18 bool `json:"over_18"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type CResp struct {
	Before string
	After  string
	// The total in the response. Not the parsed amount.
	Total int

	Images []ImageRow
}

func imgRow(base ImageRow, img RedditImage) ImageRow {
	base.Image = html.UnescapeString(img.URL())
	base.Width = img.Width()
	base.Height = img.Height()

	return base
}

func fetchPage(sub, sort, after, before string) (*CResp, error) {
	link := fmt.Sprintf("https://www.reddit.com/r/%v/%v.json?limit=100", sub, sort)
	if sort == "top" {
		link += "&t=all"
	}
	if after != "" {
		link += "&after=" + after
	}
	if before != "" {
		link += "&before=" + before
	}

	req, _ := http.NewRequest("GET", link, nil)
	req.Header.Set("User-Agent", conf.HttpStuff.UserAgent)
	if conf.HttpStuff.Credentials != "" {
		req.Header.Set("Authorization", conf.HttpStuff.Credentials)
	}

	var resp *http.Response
	retries := 0

	for {
		var err error
		log.Debug("Fetching '%v'", link)
		resp, err = http.DefaultClient.Do(req)

		if log.ErrorIfErr(err, "fetching '%v'", link) || resp.StatusCode != 200 {
			retries++

			if retries > 5 {
				if err == nil {
					err = fmt.Errorf("Rate limit")
				}

				return nil, err
			}

			if resp != nil {
				if resp.StatusCode == 429 {
					log.Debug("Rate limited")
					time.Sleep(2 * time.Minute)
				} else {
					log.Warn("Unknown status code: %v", resp.StatusCode)
				}
			}

			continue
		}

		break
	}

	o := Resp{}

	err := json.NewDecoder(resp.Body).Decode(&o)
	if err != nil {
		return nil, err
	}

	output := &CResp{
		Total:  o.Data.Total,
		Images: []ImageRow{},
	}

	subCfg := conf.Subs[sub]

	for _, p := range o.Data.Children {
		if p.Kind != "t3" {
			continue
		}
		base := ImageRow{
			PostID:    p.Data.Name,
			IsNSFW:    p.Data.Over18 || p.Data.Thumbnail == "nsfw",
			Subreddit: subCfg.Alias,
			TrueSub:   sub,
			CreatedAt: int(math.Floor(p.Data.CreatedAt)),
		}

		// Ignore new ones, awaiting review/reports <3
		if time.Since(time.Unix(int64(base.CreatedAt), 0)) <= 4*time.Hour {
			continue
		}

		if base.IsNSFW && !*subCfg.SaveNSFW {
			continue
		}

		if len(p.Data.MediaMetadata) != 0 {
			for _, img := range p.Data.MediaMetadata {
				output.Images = append(output.Images, imgRow(base, img))
			}
		} else if len(p.Data.Preview.Images) != 0 {
			pImages := p.Data.Preview.Images
			output.Images = append(output.Images, imgRow(base, pImages[len(pImages)-1]))
		}
	}

	if o.Data.After != nil {
		output.After = *o.Data.After
	}
	if o.Data.Before != nil {
		output.Before = *o.Data.Before
	}

	log.Debug("Fetched '%v'", link)

	return output, nil
}

var stopJobs = [](chan bool){}

func InitialData() {
	d := 0

	for sub, c := range conf.Subs {
		curAmt := 0
		err := db.QueryRowID(`SELECT COUNT(*) FROM images WHERE true_sub = $1`, sub, &curAmt)
		log.FatalIfErr(err, "fetching initial count data")

		if curAmt == 0 {
			log.Success("Creating initial data")
			fetchInitialData(sub)
		} else {
			log.Debug("Updating since last")
			fetchSinceLast(sub)
		}

		closer := make(chan bool)
		stopJobs = append(stopJobs, closer)

		go func(d int, closer chan bool, h time.Duration, sub string) {
			timerTillActivate := time.NewTimer(time.Duration(d) * time.Minute)
			
			select {
			case <- closer:
				return
			case <- timerTillActivate.C:
			}

			t := time.NewTicker(h * time.Hour)

			for {
				select {
				case <-t.C:
					fetchSinceLast(sub)
				case <-closer:
					return
				}
			}
		}(d, closer, time.Duration(c.Hydrate), sub)

		d += conf.Server.RefreshPad
	}
}

func insertImages(images []ImageRow) error {
	ins := [][]any{}

	for _, v := range images {
		ins = append(ins, []any{v.PostID, v.Image, v.IsNSFW, v.Subreddit, v.TrueSub, v.Width, v.Height, v.CreatedAt})
	}

	_, err := db.Insert(`images`, []string{"post_id", "img", "nsfw", "sub", "true_sub", "width", "height", "created_at"}, ins)
	return err
}

func fetchInitialData(sub string) {
	after := ""

	for {
		resp, err := fetchPage(sub, "top", after, "")
		log.FatalIfErr(err, "fetching initial data (r/%v) (after: '%v')", sub, after)

		err = insertImages(resp.Images)
		log.FatalIfErr(err, "inserting while fetching initial data")

		if resp.Total == 0 && after == "" {
			log.Fatal("Subreddit '%v' has no posts!", sub)
		}

		if resp.Total != 100 {
			return
		}

		after = resp.After
	}
}

func fetchSinceLast(sub string) {
	before := ""

	err := db.QueryRowID(`SELECT post_id FROM images WHERE true_sub = $1 ORDER BY created_at DESC LIMIT 1`, sub, &before)
	if log.ErrorIfErr(err, "fetching last post id from r/%v", sub) {
		return
	}

	for {
		resp, err := fetchPage(sub, "new", "", before)
		if log.ErrorIfErr(err, "fetching since last post (r/%v) (before: '%v')", sub, before) {
			return
		}

		err = insertImages(resp.Images)
		if log.ErrorIfErr(err, "inserting images into db during fetch since last") {
			return
		}
		if resp.Total != 100 {
			return
		}

		before = resp.Before
	}
}
