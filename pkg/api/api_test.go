package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/go-test/deep"
	"github.com/spf13/viper"
	gock "gopkg.in/h2non/gock.v1"
)

const (
	endpointYaml = `datasources:
  test:
    url: https://example.tld/v1/foo
    method: GET`
)

var (
	testEndpoint = &Endpoint{URL: "https://example.tld/v1/foo", Method: "GET"}
)

func TestEndpointUnmarshal(t *testing.T) {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	err := cfg.ReadConfig(bytes.NewBufferString(endpointYaml))
	if err != nil {
		t.Errorf("unable to read config with viper: %v", err)
	}

	e := &Endpoint{}
	err = cfg.UnmarshalKey("datasources.test", e)
	if err != nil {
		t.Errorf("unable to unmarshal endpoint config: %v", err)
	}

	if diff := deep.Equal(e, testEndpoint); len(diff) > 0 {
		t.Errorf("Endpoint doesn't match expected value:")
		for _, l := range diff {
			t.Errorf(l)
		}
	}
}

func TestEndpointMapUnmarshal(t *testing.T) {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	err := cfg.ReadConfig(bytes.NewBufferString(endpointYaml))
	if err != nil {
		t.Errorf("unable to read config with viper: %v", err)
	}

	em := make(EndpointMap)
	err = cfg.UnmarshalKey("datasources", &em)
	if err != nil {
		t.Errorf("unable to unmarshal endpoint config: %v", err)
	}

	e, ok := em["test"]
	if !ok {
		t.Errorf("unable to get test endpoint from endpoint map")
	}

	if diff := deep.Equal(e, testEndpoint); len(diff) > 0 {
		t.Errorf("Endpoint doesn't match expected value:")
		for _, l := range diff {
			t.Errorf(l)
		}
	}
}

func TestAPICall(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	e := &Endpoint{URL: "https://api.local/v1/foo", Method: http.MethodGet}
	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	data, err := e.Call("", "")
	if err != nil {
		t.Errorf("API call failed: %v", err)
	}

	if data == nil {
		t.Errorf("no data returned from API call")
	}

	fooval, ok := data["foo"]
	if !ok {
		t.Errorf("no value for key foo")
	}

	if fooval.(string) != "bar" {
		t.Errorf("wrong value returned for foo: %v", fooval)
	}
}

func TestAPIMapCall(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	e := EndpointMap{"test": &Endpoint{URL: "https://api.local/v1/foo", Method: http.MethodGet}}
	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	data, err := e.Call("test", "", "")
	if err != nil {
		t.Errorf("API call failed: %v", err)
	}

	if data == nil {
		t.Errorf("no data returned from API call")
	}

	fooval, ok := data["foo"]
	if !ok {
		t.Errorf("no value for key foo")
	}

	if fooval.(string) != "bar" {
		t.Errorf("wrong value returned for foo: %v", fooval)
	}
}
