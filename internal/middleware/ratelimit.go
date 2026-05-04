package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*ipEntry
	rate     rate.Limit
	burst    int
	interval time.Duration
}

func NewIPRateLimiter(reqPerMin, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		entries:  make(map[string]*ipEntry),
		rate:     rate.Every(time.Minute / time.Duration(reqPerMin)),
		burst:    burst,
		interval: 10 * time.Minute,
	}
}

func ClientIP(r *http.Request) string {
	return clientIP(r)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}

	trustedProxy := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_IP"))
	if trustedProxy != "" && host == trustedProxy {
		xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
		if xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
	}
	return host
}

func (l *IPRateLimiter) get(ip string) *rate.Limiter {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, entry := range l.entries {
		if now.Sub(entry.lastSeen) > l.interval {
			delete(l.entries, key)
		}
	}
	if e, ok := l.entries[ip]; ok {
		e.lastSeen = now
		return e.limiter
	}
	limiter := rate.NewLimiter(l.rate, l.burst)
	l.entries[ip] = &ipEntry{limiter: limiter, lastSeen: now}
	return limiter
}

func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.get(clientIP(r)).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
