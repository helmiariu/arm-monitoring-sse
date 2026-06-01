package collector

import (
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

// GetCPUUsage menghitung persentase beban CPU dengan sampling 500ms
func GetCPUUsage() float64 {
	percentages, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil || len(percentages) == 0 {
		return 0.0
	}
	return math.Round(percentages[0]*100) / 100
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
