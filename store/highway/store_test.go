package highway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

const (
	testHash    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testQueueId = "0123456789abcdef0123456789abcdef"
)

func testStore(t *testing.T, handler http.HandlerFunc) (*StoreClient, *[]string) {
	t.Helper()
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/center-api/transfers.php" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing bearer token")
		}
		requests = append(requests, r.Method+" "+r.URL.RawQuery)
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return NewStoreClient(&StoreClientConfig{BaseURL: server.URL + "/center-api", HTTPClient: server.Client()}), &requests
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func ctx() store.Ctx {
	return store.Ctx{APIKey: "token"}
}

const completedTransfer = `{"id":"11","service":"torrent","name":"Show S01","state":"completed","progress":100,"size":300,
	"hash":"` + testHash + `","added_at":1700000000,"files":[
		{"id":"f1","index":0,"path":"Show S01/E01.mkv","size":100,"partial":false},
		{"id":"f2","index":null,"path":"Show S01/Sub/E02.mkv","size":150,"partial":false},
		{"id":"f3","index":2,"path":"Show S01/E03.mkv.part","size":50,"partial":true}]}`

func TestGetUser(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"username":"tester","user_id":5,"plan":"Pro","premium":true}`)
	})
	user, err := s.GetUser(&store.GetUserParams{Ctx: ctx()})
	if err != nil {
		t.Fatal(err)
	}
	if user.Id != "5" || user.SubscriptionStatus != store.UserSubscriptionStatusPremium || !user.HasUsenet {
		t.Fatalf("user = %+v", user)
	}
	if (*requests)[0] != "GET stats=1" {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestGetMagnetMapsFiles(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, completedTransfer)
	})
	m, err := s.GetMagnet(&store.GetMagnetParams{Ctx: ctx(), Id: "11"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Id != "11" || m.Hash != testHash || m.Status != store.MagnetStatusDownloaded || m.AddedAt.Unix() != 1700000000 {
		t.Fatalf("magnet = %+v", m)
	}
	if len(m.Files) != 2 {
		t.Fatalf("files = %+v (partial file must be skipped)", m.Files)
	}
	if f := m.Files[0]; f.Idx != 0 || f.Path != "/E01.mkv" || f.Name != "E01.mkv" || f.Size != 100 {
		t.Fatalf("file 0 = %+v", f)
	}
	if f := m.Files[1]; f.Idx != -1 || f.Path != "/Sub/E02.mkv" || f.Name != "E02.mkv" {
		t.Fatalf("file 1 = %+v", f)
	}
	tId, fId, err := LockedFileLink(m.Files[1].Link).Parse()
	if err != nil || tId != "11" || fId != "f2" {
		t.Fatalf("link = %q -> %q %q %v", m.Files[1].Link, tId, fId, err)
	}
	if !strings.Contains((*requests)[0], "id=11") || !strings.Contains((*requests)[0], "service=torrent") {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestQueuedMagnetIsFoundByHash(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodPost:
			writeJSON(w, 200, `{"id":"`+testQueueId+`","service":"torrent","state":"queued"}`)
		case q.Get("id") == testQueueId:
			writeJSON(w, 404, `{"error":"not_found","message":"Transfer not found."}`)
		case q.Get("check") != "":
			writeJSON(w, 200, `{"service":"torrent","results":{"`+testHash+`":{"known":true,"ready":true,"id":"11","name":"Show S01","size":300}}}`)
		case q.Get("id") == "11":
			writeJSON(w, 200, completedTransfer)
		default:
			writeJSON(w, 404, `{"error":"not_found","message":"x"}`)
		}
	})

	added, err := s.AddMagnet(&store.AddMagnetParams{Ctx: ctx(), Magnet: "magnet:?xt=urn:btih:" + testHash + "&dn=Show"})
	if err != nil {
		t.Fatal(err)
	}
	if added.Id != testQueueId+"-"+testHash || added.Status != store.MagnetStatusQueued || added.Hash != testHash {
		t.Fatalf("added = %+v", added)
	}

	m, err := s.GetMagnet(&store.GetMagnetParams{Ctx: ctx(), Id: added.Id})
	if err != nil {
		t.Fatal(err)
	}
	if m.Id != "11" || m.Status != store.MagnetStatusDownloaded || len(m.Files) != 2 {
		t.Fatalf("magnet = %+v", m)
	}
	if !strings.HasPrefix((*requests)[0], "POST") {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestAddKnownMagnetReturnsFiles(t *testing.T) {
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil || !strings.Contains(r.PostForm.Get("magnet"), testHash) {
				t.Errorf("magnet form = %v (%v)", r.PostForm, err)
			}
			writeJSON(w, 200, `{"id":"11","service":"torrent","state":"completed"}`)
			return
		}
		writeJSON(w, 200, completedTransfer)
	})
	added, err := s.AddMagnet(&store.AddMagnetParams{Ctx: ctx(), Magnet: testHash})
	if err != nil {
		t.Fatal(err)
	}
	if added.Id != "11" || added.Name != "Show S01" || added.Size != 300 || len(added.Files) != 2 || added.Status != store.MagnetStatusDownloaded {
		t.Fatalf("added = %+v", added)
	}
}

func TestListMagnetsPages(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"transfers":[
			{"id":"`+testQueueId+`","service":"torrent","name":"Queued","state":"queued","size":0,"hash":"`+testHash+`","added_at":1700000100},
			{"id":"11","service":"torrent","name":"Done","state":"completed","size":300,"hash":"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB","added_at":1700000000},
			{"id":"12","service":"torrent","name":"Broken","state":"error","size":0,"hash":null,"added_at":null}],
			"total":7,"limit":3,"offset":3}`)
	})
	list, err := s.ListMagnets(&store.ListMagnetsParams{Ctx: ctx(), Limit: 3, Offset: 3})
	if err != nil {
		t.Fatal(err)
	}
	if list.TotalItems != 7 || len(list.Items) != 3 {
		t.Fatalf("list = %+v", list)
	}
	if list.Items[0].Id != testQueueId+"-"+testHash || list.Items[0].Status != store.MagnetStatusQueued {
		t.Fatalf("item 0 = %+v", list.Items[0])
	}
	if list.Items[1].Hash != strings.Repeat("b", 40) || list.Items[1].Status != store.MagnetStatusDownloaded {
		t.Fatalf("item 1 = %+v", list.Items[1])
	}
	if list.Items[2].Status != store.MagnetStatusFailed || !list.Items[2].AddedAt.IsZero() {
		t.Fatalf("item 2 = %+v", list.Items[2])
	}
	q := (*requests)[0]
	if !strings.Contains(q, "limit=3") || !strings.Contains(q, "offset=3") || !strings.Contains(q, "service=torrent") {
		t.Fatalf("request = %s", q)
	}
}

func TestRemoveMagnetReasons(t *testing.T) {
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("id") {
		case "11":
			writeJSON(w, 200, `{"removed":true}`)
		case "12":
			writeJSON(w, 200, `{"removed":false,"reason":"not_owner"}`)
		default:
			writeJSON(w, 200, `{"removed":false,"reason":"not_found"}`)
		}
	})
	if res, err := s.RemoveMagnet(&store.RemoveMagnetParams{Ctx: ctx(), Id: "11"}); err != nil || res.Id != "11" {
		t.Fatalf("remove 11 = %+v, %v", res, err)
	}
	var serr *core.StoreError
	if _, err := s.RemoveMagnet(&store.RemoveMagnetParams{Ctx: ctx(), Id: "12"}); !errors.As(err, &serr) || serr.Code != core.ErrorCodeForbidden {
		t.Fatalf("remove 12 error = %v", err)
	}
	if _, err := s.RemoveMagnet(&store.RemoveMagnetParams{Ctx: ctx(), Id: "13"}); !errors.As(err, &serr) || serr.Code != core.ErrorCodeNotFound {
		t.Fatalf("remove 13 error = %v", err)
	}
	if _, err := s.RemoveMagnet(&store.RemoveMagnetParams{Ctx: ctx(), Id: "../x"}); err == nil {
		t.Fatal("invalid id accepted")
	}
}

func TestGenerateLink(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("file") == "missing" {
			writeJSON(w, 404, `{"error":"file_not_found","message":"File not found."}`)
			return
		}
		writeJSON(w, 200, `{"id":"11","state":"completed","file":{"id":"f1","index":0,"path":"Show S01/E01.mkv","size":100,"partial":false,"download_url":"https://dl.example/signed"}}`)
	})
	link, err := s.GenerateLink(&store.GenerateLinkParams{Ctx: ctx(), Link: LockedFileLink("").Create("11", "f1")})
	if err != nil {
		t.Fatal(err)
	}
	if link.Link != "https://dl.example/signed" {
		t.Fatalf("link = %+v", link)
	}
	if q := (*requests)[0]; !strings.Contains(q, "file=f1") || !strings.Contains(q, "id=11") {
		t.Fatalf("request = %s", q)
	}

	_, err = s.GenerateLink(&store.GenerateLinkParams{Ctx: ctx(), Link: LockedFileLink("").Create("11", "missing")})
	var uerr *core.UpstreamError
	if !errors.As(err, &uerr) || uerr.Code != core.ErrorCodeNotFound || uerr.StatusCode != 404 {
		t.Fatalf("missing file error = %v", err)
	}
	if _, err := s.GenerateLink(&store.GenerateLinkParams{Ctx: ctx(), Link: "https://example.com/x"}); err == nil {
		t.Fatal("foreign link accepted")
	}
}

func TestErrorMapping(t *testing.T) {
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, `{"error":"invalid_token","message":"Invalid token."}`)
	})
	_, err := s.GetUser(&store.GetUserParams{Ctx: ctx()})
	var uerr *core.UpstreamError
	if !errors.As(err, &uerr) || uerr.Code != core.ErrorCodeUnauthorized || uerr.StatusCode != 401 {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckHashesBatches(t *testing.T) {
	batches := 0
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		batches++
		hashes := strings.Split(r.URL.Query().Get("check"), ",")
		body := `{"service":"torrent","results":{`
		for i, h := range hashes {
			if i > 0 {
				body += ","
			}
			body += `"` + h + `":{"known":true,"ready":true,"id":null,"name":"x","size":1}`
		}
		writeJSON(w, 200, body+`}}`)
	})
	hashes := make([]string, 150)
	for i := range hashes {
		hashes[i] = strings.Repeat("0", 37) + string(rune('a'+i%6)) + string(rune('a'+i/6%6)) + string(rune('a'+i/36%6))
	}
	results, err := s.checkHashes(ctx(), "torrent", hashes)
	if err != nil {
		t.Fatal(err)
	}
	if batches != 2 || len(results) != 150 || !results[hashes[149]].Ready {
		t.Fatalf("batches = %d, results = %d", batches, len(results))
	}
}
