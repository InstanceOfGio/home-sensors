package api

import (
	"crypto/subtle"
	"net/http"
)

func basicAuth(user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqUser, reqPass, ok := r.BasicAuth()
			userOK := subtle.ConstantTimeCompare([]byte(reqUser), []byte(user)) == 1
			passOK := subtle.ConstantTimeCompare([]byte(reqPass), []byte(pass)) == 1

			if !ok || !userOK || !passOK {
				w.Header().Set("WWW-Authenticate", `Basic realm="home-sensors"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
