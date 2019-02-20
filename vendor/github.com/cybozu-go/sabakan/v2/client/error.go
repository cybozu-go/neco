package client

import (
	"net/http"
)

type httpError struct {
	code   int
	reason string
}

func (e *httpError) Error() string {
	return http.StatusText(e.code) + ": " + e.reason
}

// Is4xx returns true if err contains 4xx status code
func Is4xx(err error) bool {
	err2, ok := err.(*httpError)
	if !ok {
		return false
	}
	return 400 <= err2.code && err2.code < 500
}

// IsNotFound returns true if err contains 404 status code
func IsNotFound(err error) bool {
	err2, ok := err.(*httpError)
	if !ok {
		return false
	}
	return err2.code == http.StatusNotFound
}

// IsConflict returns true if err contains 409 status code
func IsConflict(err error) bool {
	err2, ok := err.(*httpError)
	if !ok {
		return false
	}
	return err2.code == http.StatusConflict
}

// Is5xx returns true if err contains 5xx status code
func Is5xx(err error) bool {
	err2, ok := err.(*httpError)
	if !ok {
		return false
	}
	return 500 <= err2.code && err2.code < 600
}
