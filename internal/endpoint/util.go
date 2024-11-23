package endpoint

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type StoreResponse[D interface{}] struct {
	Data  D           `json:"data,omitempty"`
	Error *core.Error `json:"error,omitempty"`
}

func SendError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	var e core.StremThruError
	if sterr, ok := err.(core.StremThruError); ok {
		e = sterr
	} else {
		e = &core.Error{Cause: err}
	}

	e.Pack()

	response := &StoreResponse[any]{}
	response.Error = e.GetError()

	w.WriteHeader(e.GetStatusCode())

	log.Printf("request error: %v\n", e)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode json %v\n", e)
	}
}

func SendResponse[D interface{}](w http.ResponseWriter, statusCode int, data D, error error) {
	if error != nil {
		SendError(w, error)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := &StoreResponse[D]{}
	response.Data = data

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode json %v\n", error)
	}
}

func IsMethod(r *http.Request, method string) bool {
	return r.Method == method
}

func ReadJSONPayload[T interface{}](r *http.Request, payload T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ErrorUnsupportedMediaType(r)
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err == nil {
		return err
	}
	if err == io.EOF {
		return ErrorBadRequest(r, "missing body")
	}
	error := core.NewAPIError("failed to decode body")
	error.Cause = err
	return error
}

func getQueryInt(queryParams url.Values, name string, defaultValue int) (int, error) {
	if qVal, ok := queryParams[name]; ok {
		v := qVal[0]
		if v == "" {
			return defaultValue, nil
		}

		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, errors.New("invalid " + name)
		}
		return val, nil
	}
	return defaultValue, nil
}
