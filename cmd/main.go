package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func getHealthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /healthz request")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Status: OK")
}

func setupServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", getHealthz)

	server := &http.Server{
		Addr:    ":8001",
		Handler: mux,
	}

	return server
}

func main() {
	server := setupServer()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
