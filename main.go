package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/shadiestgoat/log"
	"github.com/shadiestgoat/redditImgCache/db"
)

func main() {
	// Lucy's 3 step plan to success:
	//
	// 1. Connect to the DB
	// 2. Populate the db (& start the jobs that will reset the data)
	// 3. Create & host the server
	//
	// PROFIT!!!

	if len(conf.Subs) == 0 {
		log.Fatal("No subreddits defined!")
	}

	db.OpenDB(conf.Server.DB)
	log.Debug("DB Loaded")

	InitialData()
	log.Debug("Loaded data")

	r := CreateMux()

	s := &http.Server{
		Addr:    ":" + fmt.Sprint(conf.Server.Port),
		Handler: r,
	}

	go func() {
		log.Success("Server starting on port %v", conf.Server.Port)
		err := s.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP Server Error: %v", err)
		}
	}()

	closeSignal := make(chan os.Signal, 2)

	signal.Notify(closeSignal, os.Interrupt)

	<-closeSignal

	log.Debug("Shutting everything down...")

	// Stop the polling jobs
	for _, c := range stopJobs {
		c <- true
	}

	log.Debug("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	log.ErrorIfErr(s.Shutdown(ctx), "shutting down server")
	cancel()

	db.Close()

	log.Success("Closed everything!")
}
