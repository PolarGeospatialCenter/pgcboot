package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	iamsign "github.com/aws/aws-sdk-go/aws/signer/v4"
)

// Endpoint is a single API endpoint/resource
type Endpoint struct {
	URL        string `mapstructure:"url"`
	Method     string `mapstructure:"method"`
	Auth       string `mapstructure:"auth"`
	iamSession *session.Session
}

// Call the Endpoint with the provided query string and requestBody (if applicable)
func (e *Endpoint) Call(query, requestBody string) (map[string]interface{}, error) {
	u, err := url.Parse(e.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %s", e.URL)
	}
	u.RawQuery = query
	req, err := http.NewRequest(e.Method, u.String(), bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	response, err := e.makeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("unable to make http request: %v", err)
	}

	value := make(map[string]interface{})
	rawBodyData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}

	err = json.Unmarshal(rawBodyData, &value)
	return value, err
}

func (e *Endpoint) iamCredentials() *session.Session {
	if e.iamSession == nil {
		e.iamSession = session.New()
	}

	return e.iamSession
}

func (e *Endpoint) iamAuth(r *http.Request, signTime time.Time) error {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	body := bytes.NewReader(bodyBytes)

	sess := e.iamCredentials()
	region := *sess.Config.Region
	service := "execute-api"
	signer := iamsign.NewSigner(sess.Config.Credentials)
	_, err = signer.Sign(r, body, service, region, signTime)
	return err
}

func (e *Endpoint) addAuth(r *http.Request) error {
	switch e.Auth {
	case "iam":
		return e.iamAuth(r, time.Now())
	default:
		return nil
	}
}

func (e *Endpoint) makeRequest(r *http.Request) (*http.Response, error) {
	c := http.Client{}
	err := e.addAuth(r)
	if err != nil {
		return nil, fmt.Errorf("error modifying request to add authentication: %v", err)
	}
	return c.Do(r)
}

// EndpointMap maps keys to api endpoints
type EndpointMap map[string]*Endpoint

// Call the endpoint from the map with the provided arguments
func (m EndpointMap) Call(endpoint, query, requestBody string) (map[string]interface{}, error) {
	e, ok := m[endpoint]
	if !ok {
		return nil, fmt.Errorf("endpoint not found: %s", endpoint)
	}

	return e.Call(query, requestBody)
}
