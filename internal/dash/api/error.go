package dash_api

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Error struct {
	Domain       string `json:"domain,omitempty"`
	ExtendedHelp string `json:"extendedHelp,omitempty"`
	Location     string `json:"location,omitempty"`
	LocationType string `json:"locationType,omitempty"`
	Message      string `json:"message"`
	Reason       string `json:"reason,omitempty"`
	SendReport   string `json:"sendReport,omitempty"`
}

type APIError struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Errors  []Error `json:"errors"`

	Cause     error  `json:"-"`
	Method    string `json:"-"`
	Path      string `json:"-"`
	RequestId string `json:"-"`
	Type      string `json:"-"`
}

func (e APIError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e APIError) Unwrap() error {
	return e.Cause
}

func (e *APIError) WithCause(cause ...error) *APIError {
	e.Cause = errors.Join(cause...)
	return e
}

func (e *APIError) InjectRequest(r *http.Request) {
	e.RequestId = r.Header.Get("Request-ID")
	e.Method = r.Method
	e.Path = r.URL.Path
}

func (e *APIError) Send(w http.ResponseWriter, r *http.Request) {
	if len(e.Errors) == 0 {
		e.Append(Error{
			Message: e.Message,
		})
	}
	SendError(w, r, e)
}

func (e *APIError) Append(err Error) *APIError {
	e.Errors = append(e.Errors, err)
	return e
}

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func ErrorBadRequest(r *http.Request, msg string) *APIError {
	if msg == "" {
		msg = "Bad Request"
	}
	err := NewAPIError(http.StatusBadRequest, msg)
	err.InjectRequest(r)
	return err
}

func ErrorUnauthorized(r *http.Request, msg string) *APIError {
	if msg == "" {
		msg = "Unauthorized"
	}
	err := NewAPIError(http.StatusUnauthorized, msg)
	err.InjectRequest(r)
	return err
}

func ErrorForbidden(r *http.Request, msg string) *APIError {
	if msg == "" {
		msg = "Forbidden"
	}
	err := NewAPIError(http.StatusForbidden, msg)
	err.InjectRequest(r)
	return err
}

func ErrorNotFound(r *http.Request, msg string) *APIError {
	if msg == "" {
		msg = "Not Found"
	}
	err := NewAPIError(http.StatusNotFound, msg)
	err.InjectRequest(r)
	return err
}

func ErrorMethodNotAllowed(r *http.Request) *APIError {
	err := NewAPIError(http.StatusMethodNotAllowed, "Method Not Allowed")
	err.InjectRequest(r)
	return err
}

func ErrorInternalServerError(r *http.Request, msg string) *APIError {
	if msg == "" {
		msg = "Internal Server Error"
	}
	err := NewAPIError(http.StatusInternalServerError, msg)
	err.InjectRequest(r)
	return err
}

func ErrorUnsupportedMediaType(r *http.Request) *APIError {
	err := NewAPIError(http.StatusUnsupportedMediaType, "Unsupported Media Type")
	err.InjectRequest(r)
	return err
}
