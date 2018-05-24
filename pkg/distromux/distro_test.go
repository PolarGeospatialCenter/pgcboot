package distromux

import (
	"log"
	"testing"

	"github.com/gorilla/mux"
)

func TestDistroMux(t *testing.T) {
	r := mux.NewRouter()
	m, err := NewDistroMux("../../test/data/branch/basic", r)
	if err != nil {
		t.Errorf("Error creating distromux: %v", err)
	}

	// generate list of routes
	routes := make(map[string]*mux.Route)
	m.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		tmpl, err := route.GetPathTemplate()
		if err != nil {
			log.Fatalf("Unable to get route path template: %v", route)
		}
		routes[tmpl] = route
		return nil
	})

	if _, ok := routes["/health"]; !ok {
		t.Fatalf("Healthcheck route not created.")
	}

	if _, ok := routes["/foo/"]; !ok {
		t.Fatalf("Foo static route not created.")
	}

	if _, ok := routes["/bar"]; !ok {
		t.Fatalf("bar template endpoint not created")
	}

	if _, ok := routes["/google/"]; !ok {
		t.Fatalf("google proxy endpoint not created")
	}
}

func TestDistroMuxTest(t *testing.T) {
	r := mux.NewRouter()
	m, err := NewDistroMux("../../test/data/branch/basic", r)
	if err != nil {
		t.Errorf("Error creating distromux: %v", err)
	}

	results, err := m.Test()
	if err != nil {
		t.Errorf("distromux tests errored: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("wrong number of test results returned: got %d, expected 1", len(results))
	}

	for p, r := range results {
		if r.Failed {
			t.Errorf("Test %s failed.", p)
			t.Error(r.Output)
		} else {
			t.Logf("Test %s Succeeded", p)
		}
	}
}
