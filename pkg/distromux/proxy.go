package distromux

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ProxyEndpoint acts as a reverse proxy to the given TargetURL
type ProxyEndpoint struct {
	TargetURL string
}

// CreateHandler returns a httputil.ReverseProxy handler
func (e *ProxyEndpoint) CreateHandler(_ string, pathPrefix string, _ map[string]interface{}) (http.Handler, error) {
	u, err := url.Parse(e.TargetURL)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = u.Host
			r.URL.Scheme = u.Scheme
			r.URL.Path = u.Path + r.URL.Path
			r.Host = u.Host
			r.RequestURI = ""
		}}
	return http.StripPrefix(pathPrefix, proxy), nil
}
