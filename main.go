package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

var httpClient = func() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	return &http.Client{
		Transport: transport,
	}
}()

func copyHeaders(src http.Header, dest http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

// ProxyHandler handles requests to the /proxy endpoint.
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "unsupported http method", http.StatusBadRequest)
		return
	}

	targetUrl := r.URL.Query().Get("url")
	if targetUrl == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	targetUrl, err := url.QueryUnescape(targetUrl)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	if u, err := url.ParseRequestURI(targetUrl); err != nil || u.Scheme == "" || u.Host == "" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	request, err := http.NewRequest(r.Method, targetUrl, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	copyHeaders(r.Header, request.Header)

	response, err := httpClient.Do(request)
	if err != nil {
		http.Error(w, "failed to request url", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	copyHeaders(response.Header, w.Header())

	w.WriteHeader(response.StatusCode)

	_, err = io.Copy(w, response.Body)
	if err != nil {
		log.Printf("stream failure: %v", err)
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/proxy", ProxyHandler)

	addr := ":" + config.Port
	log.Println("stremthru listening on " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("failed to start stremthru: %v", err)
	}
}
