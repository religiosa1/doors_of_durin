package handlers

import "net/http"

func Healthcheck(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write([]byte("ok"))
	if err == nil {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
}
