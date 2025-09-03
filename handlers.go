package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/yourname/go-shorty/internal/config"
	"github.com/yourname/go-shorty/internal/core"
	"github.com/yourname/go-shorty/internal/metrics"
)

type Router struct {
	cfg     config.Config
	svc     *core.Service
	limiter *rateLimiter
}

func NewRouter(cfg config.Config, svc *core.Service) http.Handler {
	r := chi.NewRouter()
	// Logging middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(hlog.NewHandler(log.Logger))
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, dur time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Int("size", size).
			Dur("duration", dur).
			Msg("request")
	}))
	r.Use(middleware.Recoverer)

	api := &Router{
		cfg:     cfg,
		svc:     svc,
		limiter: newRateLimiter(cfg.CreateRateRPS, cfg.CreateRateBurst),
	}

	r.MethodFunc(http.MethodGet, "/healthz", api.handleHealth)
	r.MethodFunc(http.MethodGet, "/readyz", api.handleReady)

	// Metrics
	r.MethodFunc(http.MethodGet, "/metrics", metrics.Handler)

	// Public endpoints
	r.Group(func(r chi.Router) {
		r.MethodFunc(http.MethodPost, "/api/v1/shorten", api.handleShorten)
		r.MethodFunc(http.MethodGet, "/api/v1/stats/{code}", api.handleStats)
	})

	// Redirect path
	r.MethodFunc(http.MethodGet, "/r/{code}", api.handleRedirect)

	return r
}

type shortenReq struct {
	URL  string `json:"url"`
	Code string `json:"code,omitempty"`
}

type shortenResp struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url,omitempty"`
	Target   string `json:"target"`
}

func (rt *Router) handleShorten(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !rt.limiter.Allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req shortenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	code, err := rt.svc.Shorten(strings.TrimSpace(req.Code), strings.TrimSpace(req.URL))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	target, _ := rt.svc.Resolve(code)
	resp := shortenResp{
		Code:   code,
		Target: target,
	}
	if rt.cfg.BaseURL != "" {
		resp.ShortURL = strings.TrimRight(rt.cfg.BaseURL, "/") + "/r/" + code
	}
	writeJSON(w, resp, http.StatusCreated)
	metrics.Shortens.Inc()
}

func (rt *Router) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	target, ok := rt.svc.Resolve(code)
	if !ok {
		http.NotFound(w, r)
		return
	}
	metrics.Redirects.Inc()
	metrics.RedirectByCode.WithLabelValues(code).Inc()

	go rt.svc.RecordClick(code, clientIP(r), r.UserAgent())
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func (rt *Router) handleStats(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stats, err := rt.svc.Stats(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, stats, http.StatusOK)
}

func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (rt *Router) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	// Try X-Forwarded-For or Real-IP first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if rip := r.Header.Get("X-Real-Ip"); rip != "" {
		return rip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
