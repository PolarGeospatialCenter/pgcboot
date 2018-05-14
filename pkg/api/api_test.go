package api

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"

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

func TestIAMAuth(t *testing.T) {
	e := &Endpoint{URL: "https://api.local/v1/foo", Method: http.MethodGet, Auth: "iam"}
	request, err := http.NewRequest(http.MethodGet, "https://api.local/v1/foo", bytes.NewBufferString(""))
	if err != nil {
		t.Errorf("unable to create request: %v", err)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", "asdf")
	os.Setenv("AWS_SECRET_KEY", "asdf")
	os.Setenv("AWS_REGION", "us-east-2")

	err = e.iamAuth(request, time.Unix(123456789, 0))
	if err != nil {
		t.Errorf("unable to sign request: %v", err)
	}

	if request.Header.Get("X-Amz-Date") != "19731129T213309Z" {
		t.Errorf("amazon date header doesn't match expected value: expected '19731129T213309Z'; got '%s'", request.Header.Get("X-Amz-Date"))
	}

	authz := request.Header.Get("Authorization")
	expectedAuthz := "AWS4-HMAC-SHA256 Credential=asdf/19731129/us-east-2/execute-api/aws4_request, SignedHeaders=host;x-amz-date, Signature=f8261422af8f09f27f24c0c27b9060fa3e0fc9d8a09c75d68189b542fe617385"
	if authz != expectedAuthz {
		t.Error("authorization header doesn't match expected value:")
		t.Errorf("got      %s", authz)
		t.Errorf("expected %s", expectedAuthz)
	}
}
