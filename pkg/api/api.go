package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Endpoint is a single API endpoint/resource
type Endpoint struct {
	URL    string `mapstructure:"url"`
	Method string `mapstructure:"method"`
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

	c := http.Client{}
	response, err := c.Do(req)
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
