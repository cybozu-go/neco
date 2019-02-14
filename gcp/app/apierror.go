package app

import "net/http"

// APIError is to define REST API errors.
type APIError struct {
	Status  int
	Message string
	Err     error
}

// Error implements error interface.
func (e APIError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	return e.Err.Error() + ": " + e.Message
}

// InternalServerError creates an APIError.
func InternalServerError(e error) APIError {
	return APIError{
		http.StatusInternalServerError,
		http.StatusText(http.StatusInternalServerError),
		e,
	}
}

// BadRequest creates an APIError that describes what was bad in the request.
func BadRequest(reason string) APIError {
	return APIError{http.StatusBadRequest, "invalid request: " + reason, nil}
}

// Common API errors
var (
	APIErrBadRequest     = APIError{http.StatusBadRequest, "invalid request", nil}
	APIErrForbidden      = APIError{http.StatusForbidden, "forbidden", nil}
	APIErrNotFound       = APIError{http.StatusNotFound, "requested resource is not found", nil}
	APIErrBadMethod      = APIError{http.StatusMethodNotAllowed, "method not allowed", nil}
	APIErrConflict       = APIError{http.StatusConflict, "conflicted", nil}
	APIErrLengthRequired = APIError{http.StatusLengthRequired, "content-length is required", nil}
	APIErrTooLargeAsset  = APIError{http.StatusRequestEntityTooLarge, "too large asset", nil}
)
