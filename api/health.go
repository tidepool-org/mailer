package api

import "net/http"

func LiveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
