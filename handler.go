package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"arm-monitoring-sse/collector"
)

// Broker mengelola koneksi client dan distribusi data
type Broker struct {
	clients   map[chan []byte]bool
	broadcast chan []byte
	mu        sync.Mutex
}

var broker = &Broker{
	clients:   make(map[chan []byte]bool),
	broadcast: make(chan []byte),
}

// StartBroadcaster menjalankan satu loop tunggal untuk polling data
// Panggil fungsi ini di main.go
func StartBroadcaster() {
	for {
		// 1. Ambil data HANYA SATU KALI per detik
		downSpeed, upSpeed := collector.GetNetworkSpeed()
		status := collector.ServerStatus{
			CPUUsage: collector.GetCPUUsage(),
			CPUTemp:  collector.GetCPUTemperature(),
			Download: downSpeed,
			Upload:   upSpeed,
		}

		// 2. Ubah ke JSON
		jsonData, _ := json.Marshal(status)

		// 3. Kirim ke channel broadcast
		broker.broadcast <- jsonData

		time.Sleep(1 * time.Second)
	}
}

// Menangani distribusi data ke semua klien yang terhubung
func StartDistributor() {
	for {
		msg := <-broker.broadcast
		broker.mu.Lock()
		for clientChan := range broker.clients {
			// Kirim data ke setiap channel klien
			select {
			case clientChan <- msg:
			default:
				// Jika client lambat, jangan block sistem
			}
		}
		broker.mu.Unlock()
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	// Buat channel unik untuk perangkat ini
	messageChan := make(chan []byte)

	// Daftarkan ke broker
	broker.mu.Lock()
	broker.clients[messageChan] = true
	broker.mu.Unlock()

	// Hapus pendaftaran saat koneksi ditutup
	defer func() {
		broker.mu.Lock()
		delete(broker.clients, messageChan)
		close(messageChan)
		broker.mu.Unlock()
	}()

	// Tunggu data dari broker dan kirim ke browser
	for msg := range messageChan {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
	}
}
