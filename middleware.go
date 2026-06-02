package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterInfo struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips map[string]*limiterInfo
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips: make(map[string]*limiterInfo),
		r:   r,
		b:   b,
	}
	go limiter.cleanupLoop()
	return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	info, exists := i.ips[ip]
	if !exists {
		info = &limiterInfo{
			limiter:  rate.NewLimiter(i.r, i.b),
			lastSeen: time.Now(),
		}
		i.ips[ip] = info
	} else {
		info.lastSeen = time.Now()
	}
	return info.limiter
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		i.mu.Lock()
		for ip, info := range i.ips {
			if time.Since(info.lastSeen) > 10*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

// Batasan: 2 request per detik, maksimal burst 5 request
var ipLimiter = NewIPRateLimiter(2, 5)

// Fungsi pembantu untuk mengambil IP asli di balik Cloudflare / Proxy
func getClientIP(r *http.Request) string {
	// 1. Cek jika lewat Cloudflare
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	// 2. Cek jika lewat Reverse Proxy standar (Nginx/HAProxy)
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		// X-Forwarded-For bisa berisi koma jika lewat banyak proxy, ambil yang pertama (IP asli client)
		parts := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	// 3. Fallback jika akses langsung ke IP STB tanpa proxy
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func isOriginAllowed(origin string) bool {
	primaryOrigin := os.Getenv("PRIMARY_ORIGIN")
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	cfSuffix := os.Getenv("CF_PREVIEW_SUFFIX")

	if primaryOrigin == "" {
		primaryOrigin = "http://localhost:4321"
	}

	if origin == primaryOrigin {
		return true
	}

	if allowedOriginsStr != "" {
		origins := strings.Split(allowedOriginsStr, ",")
		for _, o := range origins {
			if origin == strings.TrimSpace(o) {
				return true
			}
		}
	}

	if cfSuffix != "" && strings.HasSuffix(origin, cfSuffix) {
		return true
	}

	return false
}

func secureMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// 1. DYNAMIC CORS MATCHING
		clientOrigin := r.Header.Get("Origin")

		if clientOrigin != "" {
			if isOriginAllowed(clientOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", clientOrigin)
				// Tambahkan ini jika frontend kamu nanti butuh mengirim cookie/auth token via SSE
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				http.Error(w, "CORS Terlarang - Domain tidak diizinkan", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Tangani Preflight Request otomatis dari browser
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// 2. RATE LIMITING (Sudah anti-Cloudflare proxy)
		ip := getClientIP(r)

		limiter := ipLimiter.GetLimiter(ip)
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Too Many Requests - Santai, jangan spam!"}`))
			return
		}

		next.ServeHTTP(w, r)
	}
}
