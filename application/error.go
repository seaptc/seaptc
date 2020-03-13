package application

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	Status  int
	Message string
	Err     error
}

var (
	ErrBadRequest = &HTTPError{Status: http.StatusBadRequest}
	ErrForbidden  = &HTTPError{Status: http.StatusForbidden}
	ErrNotFound   = &HTTPError{Status: http.StatusNotFound}
)

func (e *HTTPError) Error() string {
	return fmt.Sprintf("status: %d, message: %q", e.Status, e.Message)
}
