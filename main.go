package main

import (
	"fmt"
	"github.com/cbrgm/githubevents/githubevents"
	"github.com/go-ini/ini"
	"github.com/google/go-github/v47/github"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os/exec"
)

var (
	secret, log_path, deploy_bash string
	port                          int
	cfg                           *ini.File
	logger                        *zap.Logger
	sugar                         *zap.SugaredLogger
)

func init() {
	var err error
	cfg, err = ini.Load("app.ini")
	if err != nil {
		log.Fatalf("Fail to parse 'app.ini': %v", err)
	}

	secret = cfg.Section("").Key("SECRET").MustString("")
	log_path = cfg.Section("").Key("LOG_PATH").MustString("/var/log/doge/webhook.log")
	deploy_bash = cfg.Section("").Key("DEPLOY_BASH").MustString("./deploy.sh")
	port = cfg.Section("").Key("PORT").MustInt(3421)

	log_cfg := zap.NewProductionConfig()
	log_cfg.OutputPaths = []string{
		log_path,
	}
	logger, err = log_cfg.Build()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	sugar = logger.Sugar()
}

func main() {
	// create a new event handler
	handle := githubevents.New(secret)

	// add callbacks
	handle.OnPushEventAny(
		func(deliveryID string, eventName string, e *github.PushEvent) error {
			defer sugar.Sync()
			sugar.Infow("Received PushEvent.",
				"Repo", e.Repo.Name,
				"RepoURL", e.Repo.URL,
				"Sender", e.Sender.Login,
			)
			// todo git pull & build app
			go func() {
				deploy()
			}()
			return nil
		},
	)

	// add a http handleFunc
	http.HandleFunc("/git-api/hook", func(w http.ResponseWriter, r *http.Request) {
		err := handle.HandleEventRequest(r)
		if err != nil {
			defer sugar.Sync()
			sugar.Errorln("Handle webhook fail:", err)
		}
	})

	// start the server listening on port
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil); err != nil {
		panic(err)
	}
}

func deploy() {
	defer sugar.Sync()
	sugar.Info("Start to deploy...")
	cmd := exec.Command("git", "pull", "origin", "master")
	err := cmd.Run()
	if err != nil {
		sugar.Errorln("git pull fail:", err)
	    return;
	} else {
		sugar.Info("Git pull done.")
	}
	cmd = exec.Command("go", "build", "-o", "webhook", "main.go")
	err = cmd.Run()
	if err != nil {
		sugar.Errorln("go build fail:", err)
	    return;
	} else {
		sugar.Info("Go build done.")
	}
	cmd = exec.Command(deploy_bash)
	stdoutStuderr, err := cmd.CombinedOutput()// todo kill -9 会导致无法进行下面的逻辑
	if err != nil {
		sugar.Errorln("Deploy fail:", err)
	} else {
		sugar.Infof("Finish deploy: %s", string(stdoutStuderr))
	}
}
