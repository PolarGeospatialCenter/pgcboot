package distromux

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/spf13/viper"
	gock "gopkg.in/h2non/gock.v1"
)

// DistroTestConfig describes where to find tests for the distro
type DistroTestSuite struct {
	Folder string `mapstructure:"folder"`
}

type MockDataSourceCall struct {
	DataSource string           `mapstructure:"datasource"`
	Request    MockHTTPRequest  `mapstructure:"request"`
	Response   MockHTTPResponse `mapstructure:"response"`
}

func (m *MockDataSourceCall) mock(endpoints api.EndpointMap) (gock.Mock, error) {
	source, ok := endpoints[m.DataSource]
	if !ok {
		return nil, fmt.Errorf("invalid datasource specified for mock: %s", m.DataSource)
	}
	u, err := url.Parse(source.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse datasource url %s: %v", m.DataSource, err)
	}
	req := gock.NewRequest().SetURL(u).BodyString(m.Request.Body)
	req.Method = m.Request.Method
	res := gock.NewResponse().Status(m.Response.Status).BodyString(m.Response.Body)
	return gock.NewMock(req, res), nil
}

type MockHTTPRequest struct {
	Path   string `mapstructure:"path"`
	Query  string `mapstructure:"query"`
	Body   string `mapstructure:"body"`
	Method string `mapstructure:"method"`
}

func (r *MockHTTPRequest) BuildRequest() (*http.Request, error) {
	return http.NewRequest(r.Method, "http://local"+r.Path+"?"+r.Query, bytes.NewBufferString(r.Body))
}

type MockHTTPResponse struct {
	Status int    `mapstructure:"status"`
	Body   string `mapstructure:"body"`
}

type DistroTestResult struct {
	Failed bool
	Output string
}

type DistroTestCase struct {
	InputRequest   MockHTTPRequest      `mapstructure:"request"`
	MockedData     []MockDataSourceCall `mapstructure:"mocked_data"`
	ExpectedOutput MockHTTPResponse     `mapstructure:"expected"`
}

func LoadTestCases(testsPath string) (map[string]*DistroTestCase, error) {
	testCases := make(map[string]*DistroTestCase)
	err := filepath.Walk(testsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		cfg := viper.New()
		cfg.SetConfigFile(path)
		cfg.ReadInConfig()
		testCase := &DistroTestCase{}
		err = cfg.Unmarshal(testCase)
		testCases[path] = testCase
		return err
	})
	return testCases, err
}

func (c *DistroTestCase) Test(mux *DistroMux, endpoints api.EndpointMap) *DistroTestResult {
	// Build request
	req, err := c.InputRequest.BuildRequest()
	if err != nil {
		return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to create mock request: %v", err)}
	}
	// Mock api Endpoints
	gock.DisableNetworking()
	gock.Intercept()
	defer gock.EnableNetworking()
	defer gock.Off()

	for _, mockedCall := range c.MockedData {
		mock, err := mockedCall.mock(endpoints)
		if err != nil {
			return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to create mock for data source call: %v", err)}
		}
		gock.Register(mock)
	}

	// render response
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)

	// compare response to expected
	result := &DistroTestResult{}
	var matchingBody bool
	resultBody, err := ioutil.ReadAll(response.Result().Body)
	if err != nil {
		return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to read result body: %v", err)}
	}
	matchingBody = (strings.TrimSpace(string(resultBody)) == strings.TrimSpace(c.ExpectedOutput.Body))
	result.Failed = (response.Result().StatusCode != c.ExpectedOutput.Status) || !matchingBody
	if !matchingBody {
		result.Output = fmt.Sprintf("Expected: status: %d\n%s\n", c.ExpectedOutput.Status, c.ExpectedOutput.Body) +
			fmt.Sprintf("Got:      status: %d\n%s\n", response.Result().StatusCode, string(resultBody)) +
			fmt.Sprintf("Raw Request: %v\n", req) +
			fmt.Sprintf("Raw Response: %v", response.Result())
	}
	return result
}
