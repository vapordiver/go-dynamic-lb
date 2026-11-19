package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func getHostname() string {
	hostname, _ := os.LookupEnv("INSTANCE_ID")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	return hostname
}

func getHealthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got /healthz request")

	hostname := getHostname()

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Served-By", hostname)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Status: OK\n")
	io.WriteString(w, "Host: "+hostname+"\n")
}

func simulateWork(w http.ResponseWriter, r *http.Request) {
	query, _ := url.ParseQuery(r.URL.RawQuery)

	delay, err := time.ParseDuration(query.Get("delay"))
	if err != nil {
		fmt.Println("no query detected")
		delayString, _ := os.LookupEnv("DELAY")
		if delayString == "" {
			fmt.Println("delay is not set. using default value -> 2000ms")
			delayString = "2000ms"
		}
		delay, _ = time.ParseDuration(delayString)
	}
	fmt.Println("delay: " + delay.String())

	hostname := getHostname()

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Host: "+hostname+"\n")
	case <-r.Context().Done():
		fmt.Println("Connection terminated: ", r.Context().Err())
	}
}

func setupServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", getHealthz)
	mux.HandleFunc("GET /work", simulateWork)

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
