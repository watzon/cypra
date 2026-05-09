// Package cypra contains shared SDK errors.
package cypra

import "errors"

var (
	// ErrNotFound identifies 404 responses from Cypra APIs.
	ErrNotFound = errors.New("cypra: not found")
	// ErrUnauthorized identifies 401 and 403 responses from Cypra APIs.
	ErrUnauthorized = errors.New("cypra: unauthorized")
	// ErrConflict identifies 409 responses from Cypra APIs.
	ErrConflict = errors.New("cypra: conflict")
	// ErrRateLimited identifies 429 responses from Cypra APIs.
	ErrRateLimited = errors.New("cypra: rate limited")
	// ErrValidation identifies 400 and 422 responses from Cypra APIs.
	ErrValidation = errors.New("cypra: validation failed")
)

// Error wraps a typed sentinel with the HTTP status and Cypra error code.
type Error struct {
	Err        error
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return e.Err.Error() + ": " + e.Code
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// ErrorForStatus maps an HTTP status and optional response code to a typed error.
func ErrorForStatus(status int, code string) error {
	var err error
	switch status {
	case 400, 422:
		err = ErrValidation
	case 401, 403:
		err = ErrUnauthorized
	case 404:
		err = ErrNotFound
	case 409:
		err = ErrConflict
	case 429:
		err = ErrRateLimited
	default:
		err = errors.New("cypra: api error")
	}
	return &Error{Err: err, StatusCode: status, Code: code}
}
