package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shadiestgoat/log"
	"github.com/shadiestgoat/redditImgCache/db"
)

type RespSub struct {
	NSFW map[string]int `json:"nsfw"`
	SFW  map[string]int `json:"sfw"`
}

func CreateMux() *chi.Mux {
	r := chi.NewRouter()

	if conf.Server.ExposeSubs {
		r.Get("/subs", func(w http.ResponseWriter, r *http.Request) {
			resp := RespSub{
				NSFW: map[string]int{},
				SFW:  map[string]int{},
			}

			rows, err := db.Query("SELECT sub, nsfw, COUNT(*) FROM images GROUP BY sub, nsfw")
			if log.ErrorIfErr(err, "querying for sub stats") {
				w.WriteHeader(501)
				w.Write([]byte(`{"error": "DB Error"}`))
				return
			}

			for rows.Next() {
				s, n, c := "", false, 0

				err = rows.Scan(&s, &n, &c)
				if log.ErrorIfErr(err, "scanning sub & count") {
					w.WriteHeader(501)
					w.Write([]byte(`{"error": "DB Error"}`))
					rows.Close()

					return
				}

				subs := resp.SFW

				if n {
					subs = resp.NSFW
				}

				subs[s] = c
			}

			json.NewEncoder(w).Encode(resp)
		})
	}

	r.Get(`/r/{sub}`, func(w http.ResponseWriter, r *http.Request) {
		sub := chi.URLParam(r, "sub")

		if !Aliases[sub] {
			w.WriteHeader(404)
			w.Write([]byte(`{"error": "Not Supported <3"}`))
			return
		}

		nsfwFilter := 0
		sqlNSFW := "0"

		if nsfwQueryRaw := r.URL.Query().Get("nsfw"); nsfwQueryRaw != "" {
			p, err := strconv.Atoi(nsfwQueryRaw)
			if err == nil && p >= -1 && p <= 1 {
				nsfwFilter = p
				sqlNSFW = nsfwQueryRaw
			}
		}

		possibleAnd := ""

		queryArgs := []any{sub}

		if nsfwFilter != 0 {
			possibleAnd = " AND nsfw = $2"
			queryArgs = append(queryArgs, nsfwFilter == 1)
		}

		img := ImageRow{}

		err := db.QueryRow(
			`SELECT img, nsfw, width, height FROM images WHERE sub = $1`+possibleAnd+` ORDER BY random() LIMIT 1`, queryArgs,
			&img.Image, &img.IsNSFW, &img.Width, &img.Height,
		)

		if err != nil {
			if !db.NoRows(err) {
				log.Error("Err while fetching random img: %v", err)
			}

			w.WriteHeader(501)
			w.Write([]byte(`{"error": "DB Error"}`))
			return
		}

		go db.Exec(`INSERT INTO req_stats(sub, nsfw) VALUES ($1, $2) ON CONFLICT (sub, nsfw) DO UPDATE SET requests = req_stats.requests + 1;`, sub, sqlNSFW)

		json.NewEncoder(w).Encode(img)
	})

	return r
}
