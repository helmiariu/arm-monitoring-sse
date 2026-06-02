package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"arm-monitoring-sse/collector"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Info: File .env tidak ditemukan, menggunakan environment variable dari sistem.")
	}

	// Mulai background sampling CPU
	go collector.StartCPUSampler()

	go StartBroadcaster()
	go StartDistributor()

	// Daftarkan handler dari file handler.go
	http.HandleFunc("/api/sse", secureMiddleware(statusHandler))
	http.HandleFunc("/api/testload", secureMiddleware(loadtestHandler))

	fmt.Println("Server SSE berjalan di port :4646")

	server := &http.Server{
		Addr:         ":4646",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 0, // Dibiarkan 0 agar koneksi SSE (streaming) tidak diputus timeout
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
