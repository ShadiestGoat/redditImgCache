package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shadiestgoat/log"
	"github.com/shadiestgoat/redditImgCache/db"
)

func CreateMux() *chi.Mux {
	r := chi.NewRouter()

	if conf.Server.ExposeSubs {
		r.Get("/subs", func(w http.ResponseWriter, r *http.Request) {
			subs := map[string]int{}

			rows, err := db.Query("SELECT sub, COUNT(*) FROM images GROUP BY sub")
			if log.ErrorIfErr(err, "querying for sub stats") {
				w.WriteHeader(501)
				w.Write([]byte(`{"error": "DB Error"}`))
				return
			}

			for rows.Next() {
				s, c := "", 0

				err = rows.Scan(&s, &c)
				if log.ErrorIfErr(err, "scanning sub & count") {
					w.WriteHeader(501)
					w.Write([]byte(`{"error": "DB Error"}`))
					rows.Close()

					return
				}

				subs[s] = c
			}

			json.NewEncoder(w).Encode(subs)
		})
	}

	r.Get(`/r/{sub}`, func(w http.ResponseWriter, r *http.Request) {
		sub := chi.URLParam(r, "sub")
		nsfwFilter := 0

		if nsfwQueryRaw := r.URL.Query().Get("nsfw"); nsfwQueryRaw != "" {
			p, err := strconv.Atoi(nsfwQueryRaw)
			if err == nil && p >= -1 && p <= 1 {
				nsfwFilter = p
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

		if log.ErrorIfErr(err, "fetching random img") {
			w.WriteHeader(501)
			w.Write([]byte(`{"error": "DB Error"}`))
			return
		}

		json.NewEncoder(w).Encode(img)
	})

	return r
}
