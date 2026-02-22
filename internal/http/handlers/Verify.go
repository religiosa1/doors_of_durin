package handlers

import "net/http"

type Verify struct{}

func (v Verify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
