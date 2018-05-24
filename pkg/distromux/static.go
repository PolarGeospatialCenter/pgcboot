package distromux

import (
	"net/http"
	"path/filepath"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
)

// StaticEndpoint describes configuration of endpoints that serve files.  The SourcePath is the
// relative path to the root of the tree to be served.
type StaticEndpoint struct {
	SourcePath string `mapstructure:"source"`
}

// CreateHandler ceates a handler to serve the files found at basepath/SourcePath.
func (e *StaticEndpoint) CreateHandler(basepath string, pathPrefix string, _ map[string]interface{}, _ api.EndpointMap) (http.Handler, error) {
	return http.StripPrefix(pathPrefix, http.FileServer(http.Dir(filepath.Join(basepath, e.SourcePath)))), nil
}
