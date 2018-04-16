package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/distromux"
	"github.com/gorilla/mux"
)

type DistroServer struct {
	repoPath    string
	handlers    map[string]http.Handler
	handlefuncs map[string]http.HandlerFunc
	mu          sync.Mutex
	*mux.Router
}

func NewDistroServer(repoPath string) *DistroServer {
	var s DistroServer
	s.repoPath = repoPath
	s.handlers = make(map[string]http.Handler)
	s.handlefuncs = make(map[string]http.HandlerFunc)
	s.Rebuild()
	return &s
}

func getFolders(path string) ([]string, error) {
	folders := make([]string, 0)
	base := filepath.Base(path)
	fileInfo, err := ioutil.ReadDir(path)
	if os.IsNotExist(err) {
		return folders, nil
	} else if err != nil {
		return nil, err
	}

	for _, f := range fileInfo {
		if f.IsDir() {
			folders = append(folders, filepath.Join(base, f.Name()))
		}
	}
	return folders, nil
}

// getVersionFolders returns a list of branch/tag folders within the repoPath
func (s *DistroServer) getVersionFolders() ([]string, error) {
	result := make([]string, 0)

	targetDirs := []string{"branch", "release"}
	for _, target := range targetDirs {
		folders, err := getFolders(filepath.Join(s.repoPath, target))
		if err != nil {
			return nil, err
		}
		result = append(result, folders...)
	}

	return result, nil
}

func (s *DistroServer) Rebuild() error {
	r := mux.NewRouter()
	// Walk repoPath, adding a DistroMux for each directory Found
	rebuildTime := time.Now().String()
	versionFolders, err := s.getVersionFolders()
	if err != nil {
		return err
	}

	for _, path := range versionFolders {
		srcpath := filepath.Join(s.repoPath, path)
		prefix := "/" + path + "/"
		distromux.NewDistroMux(srcpath, r.PathPrefix(prefix).Subrouter())
	}

	for p, h := range s.handlers {
		r.Handle(p, h)
	}

	for p, h := range s.handlefuncs {
		r.HandleFunc(p, h)
	}

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		statusString := fmt.Sprintf("Last updated at: %s", rebuildTime)
		w.Write([]byte(statusString))
	})

	s.mu.Lock()
	s.Router = r
	s.mu.Unlock()
	return nil
}

func (s *DistroServer) Handle(path string, h http.Handler) {
	s.handlers[path] = h
	s.Router.Handle(path, h)
}

func (s *DistroServer) HandleFunc(path string, h http.HandlerFunc) {
	s.handlefuncs[path] = h
	s.Router.HandleFunc(path, h)
}

func (s *DistroServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Distroserver got request: %v", r)
	if !s.Match(r, &mux.RouteMatch{}) {
		log.Printf("No match found for request: %s", r.URL.Path)
	}
	s.mu.Lock()
	root := s.Router
	s.mu.Unlock()
	root.ServeHTTP(w, r)
}
