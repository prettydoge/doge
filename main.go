package main

import (
	"fmt"
	"github.com/cbrgm/githubevents/githubevents"
	"github.com/go-ini/ini"
	"github.com/google/go-github/v48/github"
	"go.uber.org/zap"
	"log"
	"net/http"
)

var (
	secret string
	port   int
	cfg    *ini.File
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

func init() {
	var err error
	cfg, err = ini.Load("app.ini")
	if err != nil {
		log.Fatalf("Fail to parse 'app.ini': %v", err)
	}

	secret = cfg.Section("").Key("SECRET").MustString("")
	port = cfg.Section("").Key("PORT").MustInt(3421)

	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	sugar = zap.S()
}

func main() {
	// create a new event handler
	handle := githubevents.New("secretkey")

	// add callbacks
	handle.OnPushEventAny(
		func(deliveryID string, eventName string, e *github.PushEvent) error {
			defer logger.Sync()
			sugar.Infow("received PushEvent: %s",
				"Repo", e.Repo.Name,
				"PushID", e.PushID,
				"Action", e.Action,
				"Sender", e.Sender.Login,
			)
			// todo git pull & build app
			return nil
		},
	)

	// add a http handleFunc
	http.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
		err := handle.HandleEventRequest(r)
		if err != nil {
			defer logger.Sync()
			sugar.Errorln("handle webhook fail:", err)
		}
	})

	// start the server listening on port
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
