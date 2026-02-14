package dash_api

import (
	"net/http"

	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
)

func handleGetUsenetPoolInfo(w http.ResponseWriter, r *http.Request) {
	pool, err := usenetmanager.GetPool()
	if err != nil {
		SendError(w, r, err)
		return
	}

	info := pool.GetPoolInfo()

	SendData(w, r, 200, info)
}

func handleRebuildUsenetPool(w http.ResponseWriter, r *http.Request) {
	if _, err := usenetmanager.GetPool(); err != nil {
		SendError(w, r, err)
		return
	}

	if usenetmanager.IsPoolInUse() {
		ErrorLocked(r).WithMessage("cannot rebuild pool with active connections").Send(w, r)
		return
	}

	if err := usenetmanager.RebuildPool(); err != nil {
		SendError(w, r, err)
		return
	}

	pool, err := usenetmanager.GetPool()
	if err != nil {
		SendError(w, r, err)
		return
	}

	info := pool.GetPoolInfo()
	SendData(w, r, 200, info)
}

func AddUsenetPoolEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/usenet/pool", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetUsenetPoolInfo(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))

	router.HandleFunc("/usenet/pool/rebuild", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleRebuildUsenetPool(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
