package peer_token

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
)

func ExtractFromRequest(r *http.Request) string {
	return r.Header.Get(server.HEADER_STREMTHRU_PEER_TOKEN)
}
