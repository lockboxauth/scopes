package apiv1

import (
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"net/http"

	"darlinggo.co/api"
	yall "yall.in"

	"lockbox.dev/hmac"
	"lockbox.dev/scopes"
)

// APIv1 holds all the information that we want to
// be available for all the functions in the API,
// things like our logging, metrics, and other
// telemetry.
type APIv1 struct {
	scopes.Dependencies
	Log    *yall.Logger
	Signer hmac.Signer
}

// VerifyRequest calculates the HMAC signature of `r` and compares it to
// the passed Authorization header, while also checking the claimed SHA256
// hash of the content matches the body of the request. It either returns
// the body of the request, or a Response indicating the error in the request.
// If Response is not nil, it is meant to be returned, short-circuiting the
// request. If Response is nil, the returned string can safely be assumed to
// be an authenticated request body.
func (a APIv1) VerifyRequest(r *http.Request) (string, *Response) {
	// this is, in theory, vulnerable to replay attacks
	// but if deployed over TLS, it shouldn't matter
	var payload string
	if r.Method == "POST" || r.Method == "PUT" {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			a.Log.WithError(err).Error("error reading request")
			return "", &Response{
				Errors: api.ActOfGodError,
				Status: http.StatusInternalServerError,
			}
		}
		payload = string(body)
	}
	hash := base64.StdEncoding.EncodeToString(sha256.New().Sum([]byte(payload)))
	err := a.Signer.AuthenticateRequest(r, hash)
	if err != nil {
		a.Log.WithError(err).Debug("failed to authenticate request")
		return "", &Response{
			Errors: []api.RequestError{{
				Header: "Authorization",
				Slug:   api.RequestErrAccessDenied,
			}},
			Status: http.StatusUnauthorized,
		}
	}
	return payload, nil
}

// Response is used to encode JSON responses; it is
// the global response format for all API responses.
type Response struct {
	Scopes []Scope            `json:"scopes,omitempty"`
	Errors []api.RequestError `json:"errors,omitempty"`
	Status int                `json:"-"`
}
