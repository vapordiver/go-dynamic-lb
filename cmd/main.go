package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type api struct {
	logger *slog.Logger
	server *http.Server
	cfg    *config
}

type config struct {
	instanceID string
	port       string
	logFormat  string
	delay      time.Duration
}

func loadConfig() (*config, error) {

	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID, _ = os.Hostname()
	}

	port := os.Getenv("PORT")
	if port == "" {
		return nil, errors.New("port is not set")
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat != "text" && logFormat != "json" {
		return nil, fmt.Errorf("log_format should be set either to text or json. log_format: %v", logFormat)
	}

	delayString := os.Getenv("DELAY")
	if delayString == "" {
		delayString = "2000ms"
	}

	delay, err := time.ParseDuration(delayString)
	if err != nil {
		return nil, fmt.Errorf("could not parse delay from env: %v", err.Error())
	}

	cfg := &config{
		port:       port,
		instanceID: instanceID,
		delay:      delay,
		logFormat:  logFormat,
	}

	return cfg, nil
}

func initializeLogger(cfg *config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var logger *slog.Logger

	if cfg.logFormat == "text" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	return logger
}

func (a *api) getHealthz(w http.ResponseWriter, r *http.Request) {
	a.logger.Info("Got /healthz request")

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Served-By", a.cfg.instanceID)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Status: OK\n")
	io.WriteString(w, "Host: "+a.cfg.instanceID+"\n")
}

func (a *api) simulateWork(w http.ResponseWriter, r *http.Request) {
	a.logger.Info("Got /work request")
	delay := a.cfg.delay

	if delayString := r.URL.Query().Get("delay"); delayString != "" {
		var err error
		delay, err = time.ParseDuration(delayString)
		if err != nil || delay < 0 {
			a.logger.Warn("Incorrect query.", "endpoint", r.Pattern, "query", r.URL.RawQuery, "err", err)
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Incorrect query.\n")
			return
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Served-By", a.cfg.instanceID)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Host: "+a.cfg.instanceID+"\n")
	case <-r.Context().Done():
		a.logger.Debug("Connection terminated.", "error", r.Context().Err())
	}
}

func NewServer(logger *slog.Logger, cfg *config) *api {
	logger.Info("SERVER IS SETTING UP...")

	srv := &api{
		logger: logger,
		cfg:    cfg,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", srv.getHealthz)
	mux.HandleFunc("GET /work", srv.simulateWork)

	port := ":" + srv.cfg.port

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

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Config error.", "error", err)
		os.Exit(1)
	}

	logger := initializeLogger(cfg)

	logger = logger.With(slog.String("hostname", cfg.instanceID))

	srv := NewServer(logger, cfg)

	srv.logger.Info("Listening for connections.", "addr", srv.server.Addr)

	if err = srv.server.ListenAndServe(); err != nil {
		srv.logger.Error("Error while shutting down", "error", err)
		os.Exit(1)
	}
}
