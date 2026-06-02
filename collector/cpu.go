package collector

import (
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

var (
	cachedCPUUsage float64
	cpuUsageMu     sync.RWMutex
)

// StartCPUSampler menjalankan background sampling beban CPU setiap detik secara non-blocking
func StartCPUSampler() {
	for {
		percentages, err := cpu.Percent(500*time.Millisecond, false)
		var val float64
		if err == nil && len(percentages) > 0 {
			val = math.Round(percentages[0]*100) / 100
		}
		cpuUsageMu.Lock()
		cachedCPUUsage = val
		cpuUsageMu.Unlock()
		
		// Jeda sebelum pengambilan sampel berikutnya
		time.Sleep(500 * time.Millisecond)
	}
}

// GetCPUUsage mengembalikan persentase beban CPU secara instan & non-blocking dari cache
func GetCPUUsage() float64 {
	cpuUsageMu.RLock()
	defer cpuUsageMu.RUnlock()
	return cachedCPUUsage
}

// Membuat variabel global untuk mengingat jalur yang sukses
var cachedThermalPath string

func GetCPUTemperature() float64 {
	// 1. JIKA DI LAPTOP
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return 45.0 + rand.Float64()*4.0
	}

	// 2. JIKA PATH SUDAH PERNAH DITEMUKAN SEBELUMNYA (Langsung Tol To The Point)
	if cachedThermalPath != "" {
		return parseTempValue(cachedThermalPath)
	}

	// 3. JIKA BARU PERTAMA KALI JALAN (Cari jalurnya dulu)
	possiblePaths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/devices/virtual/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
	}

	for _, path := range possiblePaths {
		data, err := os.ReadFile(path)
		if err == nil {
			// Jalur ketemu! Simpan di variabel global agar loop ini tidak dijalankan lagi selamanya
			cachedThermalPath = path
			return convertToCelsius(data)
		}
	}

	return 0.0
}

// Helper function untuk membersihkan dan konversi data (menjaga kode tetap modular)
func convertToCelsius(data []byte) float64 {
	raw := strings.TrimSpace(string(data))
	tempInt, err := strconv.Atoi(raw)
	if err != nil {
		return 0.0
	}
	return float64(tempInt) / 1000.0
}

func parseTempValue(path string) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		// Jika suatu saat filenya mendadak hilang/error, reset cache agar sistem mencari ulang
		cachedThermalPath = ""
		return 0.0
	}
	return convertToCelsius(data)
}
