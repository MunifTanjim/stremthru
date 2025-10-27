package dash_api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/server"
)

type response struct {
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

func (res response) send(w http.ResponseWriter, r *http.Request, statusCode int) {
	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return
	}
	ctx := server.GetReqCtx(r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(res); err != nil {
		ctx.Log.Error("failed to encode json", "error", err)
	}
}

func SendError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := server.GetReqCtx(r)
	ctx.Error = err

	var e *APIError
	if !errors.As(err, &e) {
		e = ErrorInternalServerError(r, "").WithCause(err)
	}

	statusCode := e.Code
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	res := &response{Error: e}
	res.send(w, r, statusCode)
}

func SendData(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	res := &response{Data: data}
	res.send(w, r, statusCode)
}

func ReadRequestBodyJSON[T any](r *http.Request, payload T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ErrorUnsupportedMediaType(r)
	}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err == nil {
		return err
	}

	if err == io.EOF {
		return ErrorBadRequest(r, "missing body").WithCause(err)
	}

	return ErrorInternalServerError(r, "failed to decode body").WithCause(err)
}
