package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"arm-monitoring-sse/collector"
)

type Broker struct {
	clients   map[chan []byte]string
	broadcast chan []byte
	mu        sync.RWMutex // 💡 OPTIMASI: Gunakan RWMutex agar Read-Lock tidak saling antre
}

var broker = &Broker{
	clients: make(map[chan []byte]string),
	// 💡 OPTIMASI: Beri buffer 1 supaya Broadcaster tidak terblokir oleh Distributor
	broadcast: make(chan []byte, 1),
}

// GetActiveIPCount menghitung jumlah user unik berdasarkan IP
func (b *Broker) GetActiveIPCount() int {
	b.mu.RLock() // 💡 Gunakan RLock untuk membaca, jauh lebih ringan dibanding Lock biasa
	defer b.mu.RUnlock()

	// Jika hanya butuh total koneksi aktif, jalankan ini saja (jauh lebih hemat memori):
	// return len(b.clients)

	uniqueIPs := make(map[string]struct{}) // 💡 struct{} kosongan tidak memakan memori, berbeda dengan bool
	for _, ip := range b.clients {
		uniqueIPs[ip] = struct{}{}
	}
	return len(uniqueIPs)
}

type LiveDashboardStatus struct {
	collector.ServerStatus
	ActiveUsers int `json:"active_users"`
}

var (
	loadtestValue    float64
	loadtestMu       sync.Mutex
	lastLoadtestTime time.Time
)

func StartBroadcaster() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		downSpeed, upSpeed := collector.GetNetworkSpeed()

		loadtestMu.Lock()
		downSpeed += loadtestValue
		loadtestValue *= 0.7
		if loadtestValue < 0.1 {
			loadtestValue = 0
		}
		loadtestMu.Unlock()

		activeUsers := broker.GetActiveIPCount()

		status := LiveDashboardStatus{
			ServerStatus: collector.ServerStatus{
				CPUUsage: collector.GetCPUUsage(),
				CPUTemp:  collector.GetCPUTemperature(),
				Download: downSpeed,
				Upload:   upSpeed,
			},
			ActiveUsers: activeUsers,
		}

		// 💡 REKOMENDASI: Tangani error marshal (bukan di-ignore dengan '_') demi robust debugging
		jsonData, err := json.Marshal(status)
		if err != nil {
			log.Printf("Error marshal status JSON: %v", err)
			continue
		}

		broker.broadcast <- jsonData
	}
}

func StartDistributor() {
	for msg := range broker.broadcast {
		broker.mu.Lock()
		// Melakukan broadcast ke seluruh client yang terdaftar
		for clientChan := range broker.clients {
			select {
			case clientChan <- msg:
			default:
				// Client lagging/antrean penuh, skip murni tanpa membuat server macet
			}
		}
		broker.mu.Unlock()
	}
}
