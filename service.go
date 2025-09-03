package core

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/yourname/go-shorty/internal/metrics"
	"github.com/yourname/go-shorty/internal/shortid"
	"github.com/yourname/go-shorty/internal/store"
)

type Service struct {
	store        store.Store
	cache        sync.Map // code -> target
	clicksCh     chan store.ClickEvent
	clicksBufCap int
}

func NewService(s store.Store) *Service {
	return &Service{
		store:        s,
		clicksCh:     make(chan store.ClickEvent, 10000),
		clicksBufCap: 10000,
	}
}

func (s *Service) RunClickIngester(ctx context.Context) {
	for {
		select {
		case ev := <-s.clicksCh:
			if err := s.store.InsertClick(ev); err != nil {
				log.Error().Err(err).Str("code", ev.Code).Msg("insert click")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) PrewarmCache(n int) error {
	codes, err := s.store.TopCodes(n)
	if err != nil {
		return err
	}
	for _, c := range codes {
		if target, err := s.store.Get(c); err == nil {
			s.cache.Store(c, target)
		}
	}
	return nil
}

func normalizeURL(u string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("only http/https allowed")
	}
	if parsed.Host == "" {
		return "", errors.New("missing host")
	}
	return parsed.String(), nil
}

func (s *Service) Shorten(desiredCode, target string) (string, error) {
	targetNorm, err := normalizeURL(target)
	if err != nil {
		return "", err
	}
	code := desiredCode
	if code == "" {
		code = shortid.Generate(7)
	}
	if err := s.store.Save(code, targetNorm); err != nil {
		return "", err
	}
	s.cache.Store(code, targetNorm)
	return code, nil
}

func (s *Service) Resolve(code string) (string, bool) {
	if v, ok := s.cache.Load(code); ok {
		metrics.CacheHit.WithLabelValues("url").Inc()
		return v.(string), true
	}
	metrics.CacheMiss.WithLabelValues("url").Inc()
	target, err := s.store.Get(code)
	if err != nil {
		return "", false
	}
	s.cache.Store(code, target)
	return target, true
}

func (s *Service) RecordClick(code, ip, ua string) {
	select {
	case s.clicksCh <- store.ClickEvent{Code: code, IP: ip, UA: ua, Ts: time.Now()}:
	default:
		// Drop if buffer full to keep redirect fast
		metrics.ClicksDropped.Inc()
	}
}

func (s *Service) Stats(code string) (store.Stats, error) {
	return s.store.Stats(code)
}
