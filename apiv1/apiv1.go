package apiv1

import (
	"crypto/rsa"
	"io/ioutil"
	"net/http"

	"darlinggo.co/api"
	"impractical.co/auth/jose"
	"impractical.co/auth/scopes"
	yall "yall.in"
)

// APIv1 holds all the information that we want to
// be available for all the functions in the API,
// things like our logging, metrics, and other
// telemetry.
type APIv1 struct {
	scopes.Dependencies
	Log       *yall.Logger
	PublicKey rsa.PublicKey
}

func (a APIv1) VerifyRequest(r *http.Request) (string, *Response) {
	var payload string
	if r.Method == "GET" || r.Method == "DELETE" {
		payload = r.Header.Get("verification")
	} else {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return "", &Response{
				Errors: api.ActOfGodError,
				Status: http.StatusInternalServerError,
			}
		}
		payload = string(body)
	}
	parsed, err := jose.Parse(string(payload), a.PublicKey)
	if err != nil {
		return "", &Response{
			Errors: []api.RequestError{{
				Header: "Authorization",
				Slug:   api.RequestErrAccessDenied,
			}},
			Status: http.StatusUnauthorized,
		}
	}
	return parsed, nil
}

// Response is used to encode JSON responses; it is
// the global response format for all API responses.
type Response struct {
	Scopes []Scope            `json:"scopes,omitempty"`
	Errors []api.RequestError `json:"errors,omitempty"`
	Status int                `json:"-"`
}
