package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type api struct {
	logger   *slog.Logger
	hostname string
	server   *http.Server
}

func initializeLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var logger *slog.Logger

	handlerType, _ := os.LookupEnv("LOG_FORMAT")
	if handlerType == "text" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	return logger
}

func getHostname() string {
	hostname, _ := os.LookupEnv("INSTANCE_ID")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	return hostname
}

func (a *api) getHealthz(w http.ResponseWriter, r *http.Request) {
	a.logger.Info("Got /healthz request")

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Served-By", a.hostname)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Status: OK\n")
	io.WriteString(w, "Host: "+a.hostname+"\n")
}

func (a *api) simulateWork(w http.ResponseWriter, r *http.Request) {
	query, _ := url.ParseQuery(r.URL.RawQuery)

	delay, err := time.ParseDuration(query.Get("delay"))
	if err != nil {
		a.logger.Debug("No query detected.", "query", r.URL.RawQuery)
		delayString, _ := os.LookupEnv("DELAY")
		if delayString == "" {
			a.logger.Debug("Delay is not set. Using default value (2000ms)")
			delayString = "2000ms"
		}
		delay, _ = time.ParseDuration(delayString)
	}
	a.logger.Debug("Delay set.", "delay", delay.String())

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Host: "+a.hostname+"\n")
	case <-r.Context().Done():
		a.logger.Debug("Connection terminated.", "error", r.Context().Err())
	}
}

func NewServer(logger *slog.Logger, hostname string) *api {
	logger.Info("SERVER IS SETTING UP...")

	srv := &api{
		logger:   logger,
		hostname: hostname,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", srv.getHealthz)
	mux.HandleFunc("GET /work", srv.simulateWork)

	port := ":" + os.Getenv("PORT")

	srv.server = &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv
}

func main() {
	godotenv.Load()

	logger := initializeLogger()
	hostname := getHostname()

	srv := NewServer(logger.With(slog.String("hostname", hostname)), hostname)

	srv.logger.Info("Listening for connections.", "addr", srv.server.Addr)

	if err := srv.server.ListenAndServe(); err != nil {
		srv.logger.Error("Error while shutting down", "error", err)
		os.Exit(1)
	}
}
