package highway

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

const (
	testNzbMd5   = "0cc175b9c0f1b6a831c399e269772661"
	testLinkHash = "0b8e75632f44f595a021ecf266f76f7a"
)

const completedNewz = `{"id":"21","service":"usenet","name":"Show S02","state":"completed","progress":100,"size":900,
	"hash":"` + testNzbMd5 + `","added_at":1700000000,"files":[
		{"id":"n1","path":"Show S02/E01.mkv","size":800,"partial":false},
		{"id":"n2","path":"Show S02/E01.nfo","size":1,"partial":false},
		{"id":"n3","path":"Show S02/E02.mkv","size":99,"partial":true}]}`

func nzbFileHeader(t *testing.T, content string) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", "show.nzb")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(content))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["file"][0]
}

func TestCheckNewz(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"service":"usenet","results":{
			"`+testLinkHash+`":{"known":true,"ready":true,"id":"21","name":"Show S02","size":900},
			"`+testNzbMd5+`":{"known":true,"ready":false,"id":null,"name":"x","size":1}}}`)
	})
	data, err := s.CheckNewz(&store.CheckNewzParams{Ctx: ctx(), Hashes: []string{testLinkHash, testNzbMd5, "not-a-hash"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Items) != 3 || data.Items[0].Status != store.NewzStatusCached || data.Items[1].Status != store.NewzStatusUnknown || data.Items[2].Status != store.NewzStatusUnknown {
		t.Fatalf("items = %+v", data.Items)
	}
	if len(*requests) != 1 || !strings.Contains((*requests)[0], "service=usenet") || strings.Contains((*requests)[0], "not-a-hash") {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestAddNewzByLink(t *testing.T) {
	link := "https://indexer.example/api?t=get&id=abc123&apikey=SECRET"
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostForm.Get("nzb_url") != link {
			t.Errorf("nzb_url = %q", r.PostForm.Get("nzb_url"))
		}
		writeJSON(w, 200, `{"id":"`+testQueueId+`","service":"usenet","state":"queued","hash":"`+testNzbMd5+`","link_hash":"`+testLinkHash+`"}`)
	})
	data, err := s.AddNewz(&store.AddNewzParams{Ctx: ctx(), Link: link})
	if err != nil {
		t.Fatal(err)
	}
	if data.Id != testQueueId+"-"+testNzbMd5 || data.Hash != testLinkHash || data.Status != store.NewzStatusQueued {
		t.Fatalf("data = %+v", data)
	}
}

func TestAddNewzByFile(t *testing.T) {
	content := `<?xml version="1.0"?><nzb></nzb>`
	sum := md5.Sum([]byte(content))
	contentMd5 := hex.EncodeToString(sum[:])
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil || len(r.MultipartForm.File["file"]) != 1 {
			t.Errorf("missing file: %v", err)
		}
		writeJSON(w, 200, `{"id":"`+testQueueId+`","service":"usenet","state":"queued"}`)
	})
	data, err := s.AddNewz(&store.AddNewzParams{Ctx: ctx(), File: nzbFileHeader(t, content)})
	if err != nil {
		t.Fatal(err)
	}
	if data.Id != testQueueId+"-"+contentMd5 || data.Hash != contentMd5 {
		t.Fatalf("data = %+v", data)
	}
}

func TestGetNewzFindsStartedQueueEntry(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("id") == testQueueId:
			writeJSON(w, 404, `{"error":"not_found","message":"Transfer not found."}`)
		case q.Get("check") == testNzbMd5:
			writeJSON(w, 200, `{"service":"usenet","results":{"`+testNzbMd5+`":{"known":true,"ready":true,"id":"21","name":"Show S02","size":900}}}`)
		case q.Get("id") == "21" && q.Get("service") == "usenet":
			writeJSON(w, 200, completedNewz)
		default:
			t.Errorf("unexpected request %s", r.URL.RawQuery)
		}
	})
	data, err := s.GetNewz(&store.GetNewzParams{Ctx: ctx(), Id: testQueueId + "-" + testNzbMd5})
	if err != nil {
		t.Fatal(err)
	}
	if data.Id != "21" || data.Status != store.NewzStatusDownloaded || data.Hash != testNzbMd5 || len(data.Files) != 2 {
		t.Fatalf("data = %+v", data)
	}
	if data.Files[0].Name != "E01.mkv" || data.Files[0].Path != "/E01.mkv" || data.Files[0].Link != LockedFileLink("").Create("21", "n1") {
		t.Fatalf("file = %+v", data.Files[0])
	}
	if len(*requests) != 3 {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestListNewz(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"transfers":[`+completedNewz+`,{"id":"`+testQueueId+`","service":"usenet","name":"New","state":"queued","progress":0,"size":0,"hash":"`+testNzbMd5+`"}],"total":7}`)
	})
	data, err := s.ListNewz(&store.ListNewzParams{Ctx: ctx(), Limit: 2, Offset: 4})
	if err != nil {
		t.Fatal(err)
	}
	if data.TotalItems != 7 || len(data.Items) != 2 || data.Items[1].Id != testQueueId+"-"+testNzbMd5 || data.Items[1].Status != store.NewzStatusQueued {
		t.Fatalf("data = %+v", data)
	}
	if (*requests)[0] != "GET limit=2&offset=4&service=usenet" {
		t.Fatalf("requests = %v", *requests)
	}
}

func TestRemoveNewzNotOwner(t *testing.T) {
	s, _ := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Query().Get("service") != "usenet" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RawQuery)
		}
		writeJSON(w, 200, `{"removed":false,"reason":"not_owner"}`)
	})
	_, err := s.RemoveNewz(&store.RemoveNewzParams{Ctx: ctx(), Id: "21"})
	var serr *core.StoreError
	if !errors.As(err, &serr) || serr.Code != core.ErrorCodeForbidden {
		t.Fatalf("err = %v", err)
	}
}

func TestGenerateNewzLink(t *testing.T) {
	s, requests := testStore(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"id":"21","state":"completed","file":{"id":"n1","path":"Show S02/E01.mkv","size":800,"partial":false,"download_url":"https://dl.example/E01.mkv"}}`)
	})
	data, err := s.GenerateNewzLink(&store.GenerateNewzLinkParams{Ctx: ctx(), Link: LockedFileLink("").Create("21", "n1")})
	if err != nil {
		t.Fatal(err)
	}
	if data.Link != "https://dl.example/E01.mkv" || (*requests)[0] != "GET file=n1&id=21&service=usenet" {
		t.Fatalf("data = %+v, requests = %v", data, *requests)
	}
}
