package response

import (
	"net/http"
)

// SendOK always returns HTTP 200 OK
func SendOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
