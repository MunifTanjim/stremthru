package matchlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func strptr(s string) *string { return &s }

func TestResponseContract(t *testing.T) {
	valid := []string{
		`{"sid_start":null,"sid_end":null,"asid_start":null,"asid_end":null}`,
		`{"sid_start":"tt0460681:1:1","sid_end":"tt0460681:1:22","asid_start":null,"asid_end":null}`,
		`{"sid_start":null,"sid_end":null,"asid_start":"12:1","asid_end":"13:12"}`,
	}
	for _, raw := range valid {
		var r Response
		if err := r.UnmarshalJSON([]byte(raw)); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if err := r.Validate(); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
	}
	invalid := []string{
		`{}`,
		`{"sid_start":null,"sid_end":null,"asid_start":null}`,
		`{"sid_start":"tt0460681:1:2","sid_end":"tt0460681:1:1","asid_start":null,"asid_end":null}`,
		`{"sid_start":"tt0460681:1:1","sid_end":"tt1111111:1:2","asid_start":null,"asid_end":null}`,
		`{"sid_start":null,"sid_end":null,"asid_start":"notanidb:1","asid_end":null}`,
		`{"sid_start":"tt1234567:1:-1","sid_end":null,"asid_start":null,"asid_end":null}`,
	}
	for _, raw := range invalid {
		var r Response
		err := r.UnmarshalJSON([]byte(raw))
		if err == nil {
			err = r.Validate()
		}
		if err == nil {
			t.Errorf("invalid response was accepted: %s", raw)
		}
	}
}

func TestSingleFileIdentityRequiresOneCoordinate(t *testing.T) {
	movie := Response{SIDStart: strptr("tt1234567"), SIDEnd: strptr("tt1234567")}
	if got := movie.SingleSID(); got != "tt1234567" {
		t.Fatalf("movie ID: %s", got)
	}
	pack := Response{SIDStart: strptr("tt1234567:1:1"), SIDEnd: strptr("tt1234567:1:7")}
	if got := pack.SingleSID(); got != "" {
		t.Fatalf("pack must not be tagged as a single episode: %s", got)
	}
	anime := Response{ASIDStart: strptr("12:3"), ASIDEnd: strptr("12:3")}
	if got := anime.SingleASID(); got != "12:3" {
		t.Fatalf("anime ID: %s", got)
	}
}

func TestMatchLabClientUsesExternalEndpoints(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing JSON content type")
		}
		switch r.URL.Path {
		case "/match":
			fmt.Fprint(w, `{"sid_start":"tt0460681:2:3","sid_end":"tt0460681:2:3","asid_start":null,"asid_end":null}`)
		case "/parse":
			fmt.Fprint(w, `{"title":"Supernatural","year":2005,"seasons":[2],"episodes":[3],"resolution":"1080p"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 2*time.Second, 2)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Match(context.Background(), "Supernatural S02E03")
	if err != nil {
		t.Fatal(err)
	}
	if result.SingleSID() != "tt0460681:2:3" {
		t.Fatalf("unexpected SID: %+v", result)
	}
	parsed, err := client.Parse(context.Background(), "Supernatural S02E03")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Supernatural" || parsed.Year != "2005" ||
		len(parsed.Seasons) != 1 || parsed.Seasons[0] != 2 ||
		len(parsed.Episodes) != 1 || parsed.Episodes[0] != 3 {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
	if strings.Join(requests, ",") != "/match,/parse" {
		t.Fatalf("requests = %v", requests)
	}
}

func TestTransportFailuresDoNotLookLikeNoMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status":"initializing"}`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, time.Second, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Match(context.Background(), "test")
	if err == nil {
		t.Fatal("503 must remain retryable, not an empty match")
	}
}

func TestRejectUnsafeBaseURLs(t *testing.T) {
	for _, input := range []string{
		"ftp://localhost:8765", "http://user:pass@host:8765",
		"http://host:8765/?token=secret", "http://host:8765/#fragment", "http://",
	} {
		if _, err := NewClient(input, time.Second, 1); err == nil {
			t.Errorf("accepted URL %q", input)
		}
	}
}
