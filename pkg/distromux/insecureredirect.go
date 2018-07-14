package distromux

import (
	"net/http"
)

func RedirectInsecure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https" {
			h.ServeHTTP(w, r)
		} else {
			secureUrl := r.URL
			secureUrl.Host = r.Host
			secureUrl.Scheme = "https"
			http.Redirect(w, r, secureUrl.String(), http.StatusMovedPermanently)
		}
	})
}
