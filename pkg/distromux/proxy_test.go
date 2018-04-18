package distromux

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	gock "gopkg.in/h2non/gock.v1"
)

func TestProxyHTTP(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	testTarget := "https://api.local/v1/foo"
	endpoint := &ProxyEndpoint{TargetURL: "https://api.local/v1/"}
	h, err := endpoint.CreateHandler("", "/local/", nil, nil)
	if err != nil {
		t.Fatalf("Unable to create handler: %v", err)
	}

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/local/foo", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	h.ServeHTTP(response, request)
	body, err := ioutil.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatalf("Unable to read body: %v", err)
	}

	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	directRequest, err := http.NewRequest("GET", testTarget, nil)
	if err != nil {
		t.Fatalf("Unable to create direct request: %v", err)
	}

	directResponse, err := http.DefaultTransport.RoundTrip(directRequest)
	if err != nil {
		t.Fatalf("Unable to make direct request: %v", err)
	}

	directBody, err := ioutil.ReadAll(directResponse.Body)
	if err != nil {
		t.Fatalf("Unable to read direct response body: %v", err)
	}

	if response.Result().StatusCode != directResponse.StatusCode {
		t.Errorf("Proxy returned incorrect status: %d != %d", response.Result().StatusCode, directResponse.StatusCode)
	}

	if string(directBody) != string(body) {
		diff := diffmatchpatch.New()
		d := diff.DiffMain(string(directBody), string(body), false)
		t.Logf("Proxied response: %s", body)
		t.Logf("Diff: %s", diff.DiffPrettyText(d))
		t.Errorf("Direct request body doesn't match body of proxied return.")
	}
}
