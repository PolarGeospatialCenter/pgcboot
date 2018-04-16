package distromux

import (
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// Endpoint describes an interface that configuration structs should implement.
type Endpoint interface {
	CreateHandler(string, string, map[string]interface{}) (http.Handler, error)
}

// DistroConfig descibes the configuration of an instance of DistroMux
type DistroConfig struct {
	Endpoints  EndpointConfig         `mapstructure:"endpoints"`
	DistroVars map[string]interface{} `mapstructure:"vars"`
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
}

// NewDistroMux returns a new DistroMux that serves the configuration found at the supplied path
func NewDistroMux(srcpath string, router *mux.Router) *DistroMux {
	var d DistroMux
	d.basePath = srcpath
	d.Router = router
	err := d.load()
	if err != nil {
		log.Printf("An error ocurred while loading mux: %s", err)
	}
	return &d
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

func (d *DistroMux) addEndpoint(path string, endpoint Endpoint, distroVars map[string]interface{}) error {
	route := d.Router.PathPrefix(path)
	tmpl, err := route.GetPathTemplate()
	if err != nil {
		return err
	}

	h, err := endpoint.CreateHandler(d.basePath, tmpl, distroVars)
	if err != nil {
		return err
	}

	log.Printf("Creating new endpoint: %s, handler: %s", endpoint, h)
	route.Handler(h)
	return nil
}

// load populates the router object by walking through the config.
func (d *DistroMux) load() error {
	// read configuration from config directory
	config, err := d.config()
	if err != nil {
		return err
	}
	log.Printf("Read config %s", config)
	d.Router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("active"))
	})

	// add each endpoint found in the config to the mux
	for p, endpoint := range config.Endpoints.Template {
		cleanPath := path.Clean("/" + p)
		err = d.addEndpoint(cleanPath, endpoint, config.DistroVars)
		if err != nil {
			return err
		}
	}

	// add each endpoint found in the config to the mux
	for p, endpoint := range config.Endpoints.Static {
		cleanPath := path.Clean("/"+p) + "/"
		err = d.addEndpoint(cleanPath, endpoint, config.DistroVars)
		if err != nil {
			return err
		}
	}

	for p, endpoint := range config.Endpoints.Proxy {
		cleanPath := path.Clean("/"+p) + "/"
		err = d.addEndpoint(cleanPath, endpoint, config.DistroVars)
		if err != nil {
			return err
		}
	}

	return nil
}
