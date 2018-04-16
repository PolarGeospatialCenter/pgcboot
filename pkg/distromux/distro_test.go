package distromux

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestDistroMux(t *testing.T) {
	r := mux.NewRouter()
	m := NewDistroMux("../../test/data/branch/basic", r)

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

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/google/", nil)
	if err != nil {
		t.Fatalf("Unable to create request.")
	}
	r.ServeHTTP(response, request)
	if response.Result().StatusCode != http.StatusOK {
		t.Errorf("Proxy endpoint returned wrong status code: %d", response.Result().StatusCode)
		body, _ := ioutil.ReadAll(response.Result().Body)
		t.Logf("Body of bad response: %s", body)
	}
}
