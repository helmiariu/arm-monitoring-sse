package collector

import (
	"math"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

// Variabel global untuk mengingat data dari loop sebelumnya (State)
var (
	prevBytesRecv uint64
	prevBytesSent uint64
	prevTime      time.Time
)

// GetNetworkSpeed menghitung kecepatan internet (Download & Upload) dalam satuan KB/s
func GetNetworkSpeed() (float64, float64) {
	// false berarti menggabungkan semua interface (eth0, wlan0, lo) menjadi satu total
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0.0, 0.0
	}

	currentBytesRecv := counters[0].BytesRecv
	currentBytesSent := counters[0].BytesSent
	currentTime := time.Now()

	// Jika baru pertama kali aplikasi berjalan, simpan data awal dan return 0.0
	if prevTime.IsZero() {
		prevBytesRecv = currentBytesRecv
		prevBytesSent = currentBytesSent
		prevTime = currentTime
		return 0.0, 0.0
	}

	// Hitung seberapa lama jeda waktu asli yang terjadi sejak loop terakhir (dalam detik)
	duration := currentTime.Sub(prevTime).Seconds()
	if duration <= 0 {
		return 0.0, 0.0
	}

	// Hitung selisih bytes lalu konversi ke Kilobytes per Detik (KB/s)
	deltaRecv := float64(currentBytesRecv - prevBytesRecv)
	deltaSent := float64(currentBytesSent - prevBytesSent)

	downloadSpeed := (deltaRecv / 1024.0) / duration
	uploadSpeed := (deltaSent / 1024.0) / duration

	// Simpan data sekarang ke variabel global untuk modal hitungan di loop berikutnya
	prevBytesRecv = currentBytesRecv
	prevBytesSent = currentBytesSent
	prevTime = currentTime

	// Bulatkan menjadi 1 angka di belakang koma sesuai seleramu
	return math.Round(downloadSpeed*10) / 10, math.Round(uploadSpeed*10) / 10
}
