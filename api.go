package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type apiHandler struct {
	storage *inMemoryStorage

	mux *chi.Mux

	logger *zap.Logger
}

func newHandler(storage *inMemoryStorage, logger *zap.Logger) *apiHandler {
	return &apiHandler{
		mux:     chi.NewMux(),
		storage: storage,
		logger:  logger,
	}
}

func (h *apiHandler) register() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(newLoggerMiddleware(h.logger, &loggerOpts{
		WithReferer:   true,
		WithUserAgent: true,
	}))

	r.Get("/", h.index)
	r.Get("/ping", h.ping)
	r.Post("/submit", h.submit)

	h.mux.Mount("/", r)
}

func (h *apiHandler) index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi, %s", r.RemoteAddr)
}

func (h *apiHandler) ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server time: %s", time.Now())
}

func (h *apiHandler) submit(w http.ResponseWriter, r *http.Request) {
	clientAddr := r.RemoteAddr
	timeNow := time.Now()

	eventEntries := make([]eventEntry, 0)

	d := json.NewDecoder(r.Body)
	for d.More() {
		var out eventEntry

		err := d.Decode(&out)
		if err != nil {
			h.logger.Error("Can't decode part of the data", zap.Error(err))
			continue
		}

		err = out.checkRequiredFields()
		if err != nil {
			h.logger.Info("Required check failed", zap.Error(err))
			continue
		}

		out.enrichData(clientAddr, timeNow)

		eventEntries = append(eventEntries, out)
	}

	err := h.storage.addEntries(eventEntries)
	if err != nil {
		h.logger.Warn("Failed to save in memory storage", zap.Int("entries_count", len(eventEntries)), zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
