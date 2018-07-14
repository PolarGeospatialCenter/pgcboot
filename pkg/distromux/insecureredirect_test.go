package distromux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectInsecure(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://local/foo/bar?queryparam=baz", nil)
	if err != nil {
		t.Fatalf("Unable to build test request: %v", err)
	}

	response := httptest.NewRecorder()
	insecureHandler := RedirectInsecure(http.HandlerFunc(func(wr http.ResponseWriter, _ *http.Request) {
		wr.WriteHeader(http.StatusOK)
	}))

	insecureHandler.ServeHTTP(response, req)
	if response.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("Wrong status code returned: %d", response.Result().StatusCode)
	}
	location, err := response.Result().Location()
	if err != nil {
		t.Errorf("Unable to get response location: %v", err)
	}
	if location.String() != "https://local/foo/bar?queryparam=baz" {
		t.Errorf("Wrong location returned: %v", location)
	}

}

func TestRedirectInsecureXForwardedProto(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://local/foo/bar?queryparam=baz", nil)
	if err != nil {
		t.Fatalf("Unable to build test request: %v", err)
	}
	req.Header.Add("x-forwarded-proto", "https")

	response := httptest.NewRecorder()
	insecureHandler := RedirectInsecure(http.HandlerFunc(func(wr http.ResponseWriter, _ *http.Request) {
		wr.WriteHeader(http.StatusOK)
	}))

	insecureHandler.ServeHTTP(response, req)
	if response.Result().StatusCode != http.StatusOK {
		t.Errorf("Wrong status code returned: %d", response.Result().StatusCode)
	}
}
