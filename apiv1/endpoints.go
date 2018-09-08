package apiv1

import (
	"net/http"
	"strings"

	"darlinggo.co/api"
	"darlinggo.co/trout"
	yall "yall.in"
)

func (a APIv1) contextLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := a.Log.WithRequest(r).WithField("endpoint", r.Header.Get("Trout-Pattern")).WithField("method", r.Method)
		for k, v := range trout.RequestVars(r) {
			log = log.WithField("url."+strings.ToLower(k), v)
		}
		r = r.WithContext(yall.InContext(r.Context(), log))
		log.Debug("serving request")
		h.ServeHTTP(w, r)
	})
}

// Server returns an http.Handler that will handle all
// the requests for v1 of the API. The baseURL should be
// set to whatever prefix the muxer matches to pass requests
// to the Handler; consider it the root path of v1 of the API.
func (a APIv1) Server(baseURL string) http.Handler {
	var router trout.Router
	router.SetPrefix(baseURL)
	router.Endpoint("/").Methods("GET").Handler(a.contextLogger(api.NegotiateMiddleware(http.HandlerFunc(a.handleListScopes))))
	router.Endpoint("/").Methods("POST").Handler(a.contextLogger(api.NegotiateMiddleware(http.HandlerFunc(a.handleCreateScope))))
	router.Endpoint("/{id}").Methods("GET").Handler(a.contextLogger(api.NegotiateMiddleware(http.HandlerFunc(a.handleGetScope))))
	router.Endpoint("/{id}").Methods("DELETE").Handler(a.contextLogger(api.NegotiateMiddleware(http.HandlerFunc(a.handleDeleteScope))))
	router.Endpoint("/{id}").Methods("PATCH").Handler(a.contextLogger(api.NegotiateMiddleware(http.HandlerFunc(a.handleUpdateScope))))

	return router
}
