package challenge

import (
	"fmt"
	"net/http"
)

var challengeMap = map[string]string{
	"example-token": "example-token-content", // token: keyAuth
}

func challengeHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Path[len("/.well-known/acme-challenge/"):]
	response, ok := challengeMap[token]
	if !ok {
		http.NotFound(w, r)
		return
	}

	_, err := fmt.Fprint(w, response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
