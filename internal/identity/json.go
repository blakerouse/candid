// Copyright 2014 Canonical Ltd.

package identity

import (
	"net/http"

	"github.com/juju/httprequest"
	"github.com/juju/idmclient/params"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
)

// ErrLoginRequired is returned by the /debug/* endpoints when OpenID
// authentication is required.
const ErrLoginRequired params.ErrorCode = "login required"

var (
	ErrorMapper = httprequest.ErrorMapper(errToResp)
	WriteError  = ErrorMapper.WriteError
)

func errToResp(err error) (int, interface{}) {
	// Allow bakery errors to be returned as the bakery would
	// like them, so that httpbakery.Client.Do will work.
	if err, ok := errgo.Cause(err).(*httpbakery.Error); ok {
		return httpbakery.ErrorToResponse(err)
	}
	errorBody := errorResponseBody(err)
	status := http.StatusInternalServerError
	switch errorBody.Code {
	case ErrLoginRequired:
		status = http.StatusFound
	case params.ErrNotFound:
		status = http.StatusNotFound
	case params.ErrForbidden, params.ErrAlreadyExists:
		status = http.StatusForbidden
	case params.ErrBadRequest:
		status = http.StatusBadRequest
	case params.ErrUnauthorized, params.ErrNoAdminCredsProvided:
		status = http.StatusUnauthorized
	case params.ErrMethodNotAllowed:
		status = http.StatusMethodNotAllowed
	case params.ErrServiceUnavailable:
		status = http.StatusServiceUnavailable
	}

	if status == http.StatusInternalServerError {
		logger.Errorf("Internal Server Error: %s (%s)", err, errgo.Details(err))
	}

	return status, errorBody
}

// errorResponseBody returns an appropriate error response for the
// provided error.
func errorResponseBody(err error) *apiError {
	errResp := params.Error{
		Message: err.Error(),
	}
	cause := errgo.Cause(err)
	if coder, ok := cause.(errorCoder); ok {
		errResp.Code = coder.ErrorCode()
	} else if errgo.Cause(err) == httprequest.ErrUnmarshal {
		errResp.Code = params.ErrBadRequest
	}
	return &apiError{
		originalError: cause,
		Error:         errResp,
	}
}

type apiError struct {
	originalError error
	params.Error
}

func (err *apiError) SetHeader(h http.Header) {
	if setter, ok := err.originalError.(httprequest.HeaderSetter); ok {
		setter.SetHeader(h)
	}
}

type errorCoder interface {
	ErrorCode() params.ErrorCode
}
