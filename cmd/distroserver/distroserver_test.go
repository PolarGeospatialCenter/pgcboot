package main

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	gock "gopkg.in/h2non/gock.v1"
)

func TestDistroServer(t *testing.T) {
	s := NewDistroServer(path.Join("..", "..", "test", "data"))
	err := s.Rebuild()
	if err != nil {
		t.Errorf("unable to rebuild distroserver from test data: %v", err)
	}
	req, err := http.NewRequest("GET", "http://local/branch/basic/bar?id=pgc-0030", nil)
	if err != nil {
		t.Errorf("unable to create request: %v", err)
	}

	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off()
	gock.Intercept()

	gock.New("http://localhost:54321").Get("/v1/node").Reply(200).BodyString(`{"InventoryID":"pgc-0030"}`)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Result().StatusCode != 200 {
		t.Errorf("got wrong status from request: %d", w.Result().StatusCode)
	}
}
