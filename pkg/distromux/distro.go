package distromux

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type DistroVars map[string]interface{}

func (v DistroVars) Vars(_ *http.Request) DistroVars {
	log.Printf("Getting distrovars: %v", v)
	return v
}

func (v DistroVars) SetContextForRequest(r *http.Request) *http.Request {
	return r.WithContext(NewDistroVarsContext(r.Context(), v))
}

// Endpoint describes an interface that configuration structs should implement.
type Endpoint interface {
	CreateHandler(string, string, api.EndpointMap) (http.Handler, error)
}

// DistroConfig descibes the configuration of an instance of DistroMux
type DistroConfig struct {
	Endpoints   EndpointConfig  `mapstructure:"endpoints"`
	DataSources api.EndpointMap `mapstructure:"datasources"`
	Test        DistroTestSuite `mapstructure:"test"`
	DistroVars  DistroVars      `mapstructure:"vars"`
}

type EndpointConfig struct {
	Template map[string]*TemplateEndpoint `mapstructure:"template"`
	Static   map[string]*StaticEndpoint   `mapstructure:"static"`
	Proxy    map[string]*ProxyEndpoint    `mapstructure:"proxy"`
}

// DistroMux configures a gorilla/mux Router that will serve the contents of a
// folder based on a config file found in either the root of the folder, or in a
// config subdirectory.
type DistroMux struct {
	*mux.Router
	basePath string
	cfg      *DistroConfig
}

// NewDistroMux returns a new DistroMux that serves the configuration found at the supplied path
func NewDistroMux(srcpath string, router *mux.Router) (*DistroMux, error) {
	var d DistroMux
	d.basePath = srcpath
	d.Router = router
	cfg, err := d.config()
	if err != nil {
		return nil, fmt.Errorf("Failed to parse distro configuration: %v", err)
	}
	d.cfg = cfg
	err = d.load()
	if err != nil {
		return nil, fmt.Errorf("An error ocurred while loading distro folder %s: %v", d.basePath, err)
	}
	return &d, nil
}

// config parses and returns the config for this DistroMux
func (d *DistroMux) config() (*DistroConfig, error) {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.AddConfigPath(d.basePath)
	cfg.AddConfigPath(filepath.Join(d.basePath, "config"))
	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var config DistroConfig
	err = cfg.Unmarshal(&config)
	return &config, err
}

func (d *DistroMux) addEndpoint(path string, endpoint Endpoint, dataSources api.EndpointMap) error {
	route := d.Router.PathPrefix(path)
	tmpl, err := route.GetPathTemplate()
	if err != nil {
		return err
	}

	h, err := endpoint.CreateHandler(d.basePath, tmpl, dataSources)
	if err != nil {
		return err
	}

	route.Handler(h)
	return nil
}

// load populates the router object by walking through the config.
func (d *DistroMux) load() error {
	var err error
	// read configuration from config directory
	config := d.cfg
	d.Router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("active"))
	})

	// add each endpoint found in the config to the mux
	for p, endpoint := range config.Endpoints.Template {
		cleanPath := path.Clean("/" + p)
		err = d.addEndpoint(cleanPath, endpoint, config.DataSources)
		if err != nil {
			return fmt.Errorf("unable to load template endpoint %s: %v", p, err)
		}
	}

	// add each endpoint found in the config to the mux
	for p, endpoint := range config.Endpoints.Static {
		cleanPath := path.Clean("/"+p) + "/"
		err = d.addEndpoint(cleanPath, endpoint, config.DataSources)
		if err != nil {
			return fmt.Errorf("unable to load static endpoint %s: %v", p, err)
		}
	}

	for p, endpoint := range config.Endpoints.Proxy {
		cleanPath := path.Clean("/"+p) + "/"
		err = d.addEndpoint(cleanPath, endpoint, config.DataSources)
		if err != nil {
			return fmt.Errorf("unable to load proxy endpoint %s: %v", p, err)
		}
	}

	return nil
}

func (d *DistroMux) Test() (map[string]*DistroTestResult, error) {
	testConfig := d.cfg.Test
	testsFolder := testConfig.Folder
	if testsFolder == "" {
		testsFolder = "tests"
	}

	// Load test cases from folder
	testCases, err := LoadTestCases(path.Join(d.basePath, testsFolder))
	if err != nil {
		return nil, fmt.Errorf("failed loading test cases from file: %v", err)
	}

	testResults := make(map[string]*DistroTestResult)
	for p, c := range testCases {
		testResults[p] = c.Test(d, d.cfg.DataSources)
	}

	return testResults, nil
}

func (d *DistroMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := DistroVarsFromContext(r.Context()); !ok {
		r = r.WithContext(NewDistroVarsContext(r.Context(), d.cfg.DistroVars))
	}
	d.Router.ServeHTTP(w, r)
}
