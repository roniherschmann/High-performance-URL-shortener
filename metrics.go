package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Redirects = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redirect_requests_total",
		Help: "Total redirect requests.",
	})
	RedirectByCode = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "redirect_requests_by_code_total",
		Help: "Redirect requests by code.",
	}, []string{"code"})
	Shortens = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shorten_requests_total",
		Help: "Total shorten requests.",
	})
	CacheHit = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_hit_total",
		Help: "Cache hits.",
	}, []string{"kind"})
	CacheMiss = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_miss_total",
		Help: "Cache misses.",
	}, []string{"kind"})
	ClicksDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "clicks_dropped_total",
		Help: "Clicks dropped due to full buffer.",
	})
)

func init() {
	prometheus.MustRegister(Redirects, RedirectByCode, Shortens, CacheHit, CacheMiss, ClicksDropped)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
