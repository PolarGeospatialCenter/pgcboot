package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	treebuilder "github.com/PolarGeospatialCenter/pgcboot/pkg/gittree"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

func GetParameter(path string) string {
	s := ssm.New(session.New())
	out, err := s.GetParameter(&ssm.GetParameterInput{Name: aws.String(fmt.Sprintf("/distroserver/%s", path)), WithDecryption: aws.Bool(true)})
	if err != nil {
		return ""
	}
	return out.String()
}

func main() {
	// setup config
	cfg := viper.New()
	cfg.SetConfigName("distroserver")
	cfg.AddConfigPath("/etc/distroserver")
	cfg.AddConfigPath(".")
	cfg.SetDefault("tempdir", "")
	// load config
	cfg.ReadInConfig()

	// Create temporary path for repository
	repoPath, err := ioutil.TempDir(cfg.GetString("tempdir"), "repository")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(repoPath)
	log.Printf("RepositoryPath: %s", repoPath)

	// Create directory to store work trees
	treePath, err := ioutil.TempDir(cfg.GetString("tempdir"), "worktrees")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(treePath)
	log.Printf("Working tree path: %s", treePath)

	server := NewDistroServer(treePath)

	updateFunc := func(_ interface{}, _ webhooks.Header) {
		builder, err := treebuilder.NewSSHBuilder(GetParameter("gitrepourl"), GetParameter("deploykey"), treePath, repoPath)
		if err != nil {
			log.Fatalf("Unable to create git tree builder: %v", err)
		}

		err = builder.BuildGitTree()
		if err != nil {
			log.Fatalf("Unable to build git tree: %v", err)
		}

		err = server.Rebuild()
		if err != nil {
			log.Fatalf("Unable to rebuild ipxeserver: %v", err)
		}
	}

	hook := github.New(&github.Config{Secret: GetParameter("webhook_secret")})
	hook.RegisterEvents(updateFunc, github.ReleaseEvent, github.PushEvent)
	server.Handle("/updatehook", webhooks.Handler(hook))

	updateFunc(nil, nil)

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      server,
		TLSConfig:    &tls.Config{CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA256, tls.TLS_RSA_WITH_AES_128_CBC_SHA, tls.TLS_RSA_WITH_AES_256_CBC_SHA}},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	go func() {
		var err error
		if cfg.GetString("ssl.key") == "" || cfg.GetString("ssl.cert") == "" {
			err = httpServer.ListenAndServe()
		} else {
			err = httpServer.ListenAndServeTLS(cfg.GetString("ssl.cert"), cfg.GetString("ssl.key"))
		}
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Unable to serve: %s", err)
		}
	}()

	server.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		log.Printf("Route: %v", route)
		return nil
	})

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGTERM)
	var exit bool
	for exit == false {
		select {
		case signal := <-signalChan:
			switch signal {
			case syscall.SIGHUP:
				updateFunc(nil, nil)
			default:
				log.Printf("Got signal: %v", signal)
				log.Printf("Shutting down http server ...")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				err := httpServer.Shutdown(ctx)
				if err != nil {
					log.Fatalf("Error while shutting down http server: %v", err)
				}
				log.Printf("Shutdown Complete")
				exit = true
			}
		}
	}
	log.Printf("Exiting")
}
