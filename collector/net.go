package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	prevRx   uint64
	prevTx   uint64
	prevTime time.Time

	// 💡 Variabel tambahan untuk menampung cache
	cachedInterface string
	lastChecked     time.Time
	netMu           sync.Mutex
)


// 🔍 Fungsi Deteksi Otomatis dengan Fitur Cache 10 Detik
func getActiveWANInterface() string {
	// Opsi Paksa Manual lewat ENV tetap diprioritaskan
	if envIface := os.Getenv("INTERFACE"); envIface != "" {
		return envIface
	}

	netMu.Lock()
	if cachedInterface != "" && time.Since(lastChecked) < 10*time.Second {
		val := cachedInterface
		netMu.Unlock()
		return val
	}
	netMu.Unlock()

	// Bagian ini hanya dieksekusi 10 detik sekali jika cache kedaluwarsa
	file, err := os.Open("/proc/net/route")
	if err != nil {
		netMu.Lock()
		val := cachedInterface
		netMu.Unlock()
		if val != "" {
			return val // Gunakan cache lama jika file error
		}
		return "eth0"
	}
	defer file.Close()

	foundInterface := "eth0" // Fallback default
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == "00000000" {
			foundInterface = fields[0]
			break
		}
	}

	// 💾 Perbarui data cache dan waktu pengecekan terakhir
	netMu.Lock()
	cachedInterface = foundInterface
	lastChecked = time.Now()
	val := cachedInterface
	netMu.Unlock()

	return val
}

// 🚀 Fungsi Utama (Tetap berjalan setiap detik, tapi pemanggilan fungsinya jadi sangat ringan)
func GetNetworkSpeed() (float64, float64) {
	// Memanggil fungsi deteksi (yang sekarang sudah pintar & menggunakan cache)
	targetInterface := getActiveWANInterface()

	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var currentRx, currentTx uint64
	now := time.Now()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, targetInterface) {
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}
			currentRx, _ = strconv.ParseUint(fields[1], 10, 64)
			currentTx, _ = strconv.ParseUint(fields[9], 10, 64)
			break
		}
	}

	netMu.Lock()
	defer netMu.Unlock()

	if prevTime.IsZero() {
		prevRx = currentRx
		prevTx = currentTx
		prevTime = now
		return 0, 0
	}

	duration := now.Sub(prevTime).Seconds()
	if duration <= 0 {
		duration = 1
	}

	downloadSpeed := (float64(currentRx-prevRx) / duration) / 1024
	uploadSpeed := (float64(currentTx-prevTx) / duration) / 1024

	prevRx = currentRx
	prevTx = currentTx
	prevTime = now

	return downloadSpeed, uploadSpeed
}
