package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func getHealthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /healthz request")
	hostname, isHostnameSet := os.LookupEnv("INSTANCE_ID")
	if !isHostnameSet {
		hostname, _ = os.Hostname()
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Served-By", hostname)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Status: OK\n")
	io.WriteString(w, "Host: "+hostname+"\n")
}

func setupServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", getHealthz)

	port := ":" + os.Getenv("PORT")

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	return server
}

func main() {
	godotenv.Load()

	server := setupServer()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
