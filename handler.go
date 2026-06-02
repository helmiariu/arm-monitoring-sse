package main

import (
	"fmt"
	"net/http"
	"time"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	// 💡 KOREKSI UTAMA: Gunakan fungsi getClientIP dari middleware.go
	// Agar user di balik Cloudflare/Proxy terhitung sebagai IP unik yang akurat.
	ip := getClientIP(r)

	messageChan := make(chan []byte, 1)

	// Daftarkan client ke broker
	broker.mu.Lock()
	broker.clients[messageChan] = ip
	broker.mu.Unlock()

	// Bersihkan client saat koneksi terputus (tab browser ditutup)
	defer func() {
		broker.mu.Lock()
		delete(broker.clients, messageChan)
		broker.mu.Unlock()

		// 💡 CATATAN KEAMANAN: close() di sini 100% aman karena pembacaan map
		// di StartDistributor dilindungi oleh Mutex Lock yang sama.
		close(messageChan)
	}()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			// Browser client menutup koneksi atau pindah halaman
			return
		case msg, ok := <-messageChan:
			if !ok {
				return
			}
			// Kirim data dengan format standar SSE (data: <json>\n\n)
			_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func loadtestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	loadtestMu.Lock()
	defer loadtestMu.Unlock()

	// Mekanisme Cooldown: Batasi trigger loadtest maksimal 1x per 200ms
	now := time.Now()
	if now.Sub(lastLoadtestTime) < 200*time.Millisecond {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Terlalu banyak permintaan, coba lagi nanti"}`))
		return
	}
	lastLoadtestTime = now

	// Naikkan nilai loadtest simulasi hingga batasan 5000
	if loadtestValue < 5000 {
		loadtestValue += 250
		if loadtestValue > 5000 {
			loadtestValue = 5000
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"loadtest_triggered"}`))
}
