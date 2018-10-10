package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	iamsign "github.com/aws/aws-sdk-go/aws/signer/v4"
)

// APIResponse is a data structure encapsulating the return from an api endpoint
type APIResponse struct {
	Status int
	Data   interface{}
}

// Endpoint is a single API endpoint/resource
type Endpoint struct {
	URL        string `mapstructure:"url"`
	Method     string `mapstructure:"method"`
	Auth       string `mapstructure:"auth"`
	iamSession *session.Session
}

func (e *Endpoint) GetUrl(subPath, query string) (*url.URL, error) {
	tmpl, err := template.New("url").Funcs(map[string]interface{}{"env": os.Getenv}).Parse(e.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing url as template: %v", err)
	}

	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, nil)
	if err != nil {
		return nil, fmt.Errorf("error rendering url as template: %v", err)
	}

	u, err := url.Parse(wr.String())
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %s", e.URL)
	}
	u.Path = path.Join(u.Path, subPath)
	u.RawQuery = query
	return u, nil
}

// Call the Endpoint with the provided query string and requestBody (if applicable)
func (e *Endpoint) Call(subPath, query, requestBody string) (*APIResponse, error) {
	u, err := e.GetUrl(subPath, query)
	if err != nil {
		return nil, fmt.Errorf("unable to build URL (%s): %v", e.URL, err)
	}
	var body io.Reader
	switch e.Method {
	case http.MethodGet:
		body = nil
	default:
		body = bytes.NewBufferString(requestBody)
	}
	req, err := http.NewRequest(e.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	response, err := e.makeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("unable to make http request: %v", err)
	}

	apiResponse := &APIResponse{Status: response.StatusCode, Data: make(map[string]interface{})}
	rawBodyData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		apiResponse.Data.(map[string]interface{})["error"] = "unable to read api response body"
		return apiResponse, fmt.Errorf("unable to read response body: %v", err)
	}

	err = json.Unmarshal(rawBodyData, &(apiResponse.Data))
	if err != nil {
		apiResponse.Data.(map[string]interface{})["error"] = "unable to unmarshal response body"
		return apiResponse, fmt.Errorf("unable to unmarshal response body: %v -- request '%s' -- raw body '%s'", err, u.String(), string(rawBodyData))
	}
	return apiResponse, err
}

func (e *Endpoint) iamCredentials() *session.Session {
	if e.iamSession == nil || e.iamSession.Config.Credentials.IsExpired() {
		e.iamSession = session.New()
	}

	return e.iamSession
}

func (e *Endpoint) iamAuth(r *http.Request, signTime time.Time) error {
	var body io.ReadSeeker
	if r.Body != nil {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		body = bytes.NewReader(bodyBytes)
	} else {
		body = bytes.NewReader([]byte{})
	}

	sess := e.iamCredentials()
	region := *sess.Config.Region
	service := "execute-api"
	signer := iamsign.NewSigner(sess.Config.Credentials)
	_, err := signer.Sign(r, body, service, region, signTime)
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
func (m EndpointMap) Call(endpoint, subPath, query, requestBody string) (*APIResponse, error) {
	e, ok := m[endpoint]
	if !ok {
		return nil, fmt.Errorf("endpoint not found: %s", endpoint)
	}

	return e.Call(subPath, query, requestBody)
}
