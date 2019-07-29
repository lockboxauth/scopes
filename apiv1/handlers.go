package apiv1

import (
	"encoding/json"
	"net/http"

	"darlinggo.co/api"
	"darlinggo.co/trout"
	yall "yall.in"

	"lockbox.dev/scopes"
)

func (a APIv1) handleCreateScope(w http.ResponseWriter, r *http.Request) {
	input, resp := a.VerifyRequest(r)
	if resp != nil {
		api.Encode(w, r, resp.Status, resp)
		return
	}
	var body Scope
	err := json.Unmarshal([]byte(input), &body)
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Debug("Error decoding request body")
		api.Encode(w, r, http.StatusBadRequest, Response{Errors: api.InvalidFormatError})
		return
	}
	scope := coreScope(body)
	var reqErrs []api.RequestError

	// ClientPolicy must be set and valid
	if scope.ClientPolicy == "" {
		reqErrs = append(reqErrs, api.RequestError{Field: "/clientPolicy", Slug: api.RequestErrMissing})
	} else if !scopes.IsValidPolicy(scope.ClientPolicy) {
		reqErrs = append(reqErrs, api.RequestError{Field: "/clientPolicy", Slug: api.RequestErrInvalidValue})
	}

	// UserPolicy must be set and valid
	if scope.UserPolicy == "" {
		reqErrs = append(reqErrs, api.RequestError{Field: "/userPolicy", Slug: api.RequestErrMissing})
	} else if scope.UserPolicy != "" && !scopes.IsValidPolicy(scope.UserPolicy) {
		reqErrs = append(reqErrs, api.RequestError{Field: "/userPolicy", Slug: api.RequestErrInvalidValue})
	}

	if scope.ID == "" {
		reqErrs = append(reqErrs, api.RequestError{Field: "/id", Slug: api.RequestErrMissing})
	}
	if len(reqErrs) > 0 {
		api.Encode(w, r, http.StatusBadRequest, reqErrs)
		return
	}
	err = a.Storer.Create(r.Context(), scope)
	if err != nil {
		if err == scopes.ErrScopeAlreadyExists {
			api.Encode(w, r, http.StatusBadRequest, Response{Errors: []api.RequestError{{Field: "/id", Slug: api.RequestErrConflict}}})
			return
		}
		yall.FromContext(r.Context()).WithError(err).Error("Error creating scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	yall.FromContext(r.Context()).WithField("scope_id", scope.ID).Debug("scope created")
	api.Encode(w, r, http.StatusCreated, Response{Scopes: []Scope{apiScope(scope)}})
}

func (a APIv1) handleUpdateScope(w http.ResponseWriter, r *http.Request) {
	vars := trout.RequestVars(r)
	id := vars.Get("id")
	if id == "" {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrMissing}}})
		return
	}

	input, resp := a.VerifyRequest(r)
	if resp != nil {
		api.Encode(w, r, resp.Status, resp)
		return
	}

	var body Change
	err := json.Unmarshal([]byte(input), &body)
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Debug("Error decoding request body")
		api.Encode(w, r, http.StatusBadRequest, Response{Errors: api.InvalidFormatError})
		return
	}
	change := coreChange(body)
	var reqErrs []api.RequestError

	// UserPolicy must be valid if it's set
	if change.UserPolicy != nil && !scopes.IsValidPolicy(*change.UserPolicy) {
		reqErrs = append(reqErrs, api.RequestError{Field: "/userPolicy", Slug: api.RequestErrInvalidValue})
	}

	// ClientPolicy must be valid if it's set
	if change.ClientPolicy != nil && !scopes.IsValidPolicy(*change.ClientPolicy) {
		reqErrs = append(reqErrs, api.RequestError{Field: "/clientPolicy", Slug: api.RequestErrInvalidValue})
	}

	if len(reqErrs) > 0 {
		api.Encode(w, r, http.StatusBadRequest, reqErrs)
		return
	}
	err = a.Storer.Update(r.Context(), id, change)
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Error("Error updating scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	yall.FromContext(r.Context()).WithField("scope_id", id).Debug("scope updated")
	scops, err := a.Storer.GetMulti(r.Context(), []string{id})
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Error("Error retrieving scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	scope, ok := scops[id]
	if !ok {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrNotFound}}})
		return
	}
	api.Encode(w, r, http.StatusOK, Response{Scopes: []Scope{apiScope(scope)}})
}

func (a APIv1) handleGetScope(w http.ResponseWriter, r *http.Request) {
	vars := trout.RequestVars(r)
	id := vars.Get("id")
	if id == "" {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrMissing}}})
		return
	}

	input, resp := a.VerifyRequest(r)
	if resp != nil {
		api.Encode(w, r, resp.Status, resp)
		return
	}
	if input != "GET,"+id {
		api.Encode(w, r, http.StatusUnauthorized, Response{Errors: []api.RequestError{{Header: "Authorization", Slug: api.RequestErrAccessDenied}}})
		return
	}

	scops, err := a.Storer.GetMulti(r.Context(), []string{id})
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Error("Error retrieving scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	scope, ok := scops[id]
	if !ok {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrNotFound}}})
		return
	}
	api.Encode(w, r, http.StatusOK, Response{Scopes: []Scope{apiScope(scope)}})
}

func (a APIv1) handleDeleteScope(w http.ResponseWriter, r *http.Request) {
	vars := trout.RequestVars(r)
	id := vars.Get("id")
	if id == "" {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrMissing}}})
		return
	}

	input, resp := a.VerifyRequest(r)
	if resp != nil {
		api.Encode(w, r, resp.Status, resp)
		return
	}
	if input != "DELETE,"+id {
		api.Encode(w, r, http.StatusUnauthorized, Response{Errors: []api.RequestError{{Header: "Authorization", Slug: api.RequestErrAccessDenied}}})
		return
	}

	scops, err := a.Storer.GetMulti(r.Context(), []string{id})
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Error("Error retrieving scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	scope, ok := scops[id]
	if !ok {
		api.Encode(w, r, http.StatusNotFound, Response{Errors: []api.RequestError{{Param: "id", Slug: api.RequestErrNotFound}}})
		return
	}

	err = a.Storer.Delete(r.Context(), id)
	if err != nil {
		yall.FromContext(r.Context()).WithError(err).Error("Error deleting scope")
		api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
		return
	}
	yall.FromContext(r.Context()).WithField("scope_id", id).Debug("scope deleted")
	api.Encode(w, r, http.StatusOK, Response{Scopes: []Scope{apiScope(scope)}})
}

func (a APIv1) handleListScopes(w http.ResponseWriter, r *http.Request) {
	filterDefault := r.URL.Query().Get("default")
	filterIDs := r.URL.Query()["id"]

	if len(filterIDs) > 0 && filterDefault != "" {
		api.Encode(w, r, http.StatusBadRequest, Response{Errors: []api.RequestError{{Param: "default,id", Slug: api.RequestErrConflict}}})
		return
	} else if len(filterIDs) < 1 && filterDefault == "" {
		api.Encode(w, r, http.StatusBadRequest, Response{Errors: []api.RequestError{{Param: "default", Slug: api.RequestErrMissing}}})
		return
	} else if filterDefault != "" && filterDefault != "true" {
		api.Encode(w, r, http.StatusBadRequest, Response{Errors: []api.RequestError{{Param: "default", Slug: api.RequestErrInvalidValue}}})
		return
	}

	input, resp := a.VerifyRequest(r)
	if resp != nil {
		api.Encode(w, r, resp.Status, resp)
		return
	}
	if input != "LIST,"+r.URL.RawQuery {
		api.Encode(w, r, http.StatusUnauthorized, Response{Errors: []api.RequestError{{Header: "Authorization", Slug: api.RequestErrAccessDenied}}})
		return
	}

	var scops []scopes.Scope

	if len(filterIDs) > 0 {
		resp, err := a.Storer.GetMulti(r.Context(), filterIDs)
		if err != nil {
			yall.FromContext(r.Context()).WithError(err).Error("Error retrieving scopes")
			api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
			return
		}
		for _, v := range resp {
			scops = append(scops, v)
		}
	} else if filterDefault != "" {
		resp, err := a.Storer.ListDefault(r.Context())
		if err != nil {
			yall.FromContext(r.Context()).WithError(err).Error("Error retrieving scopes")
			api.Encode(w, r, http.StatusInternalServerError, Response{Errors: api.ActOfGodError})
			return
		}
		scops = append(scops, resp...)
	}
	yall.FromContext(r.Context()).Debug("scopes retrieved")
	api.Encode(w, r, http.StatusOK, Response{Scopes: apiScopes(scops)})
}
