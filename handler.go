package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	// Import sub-package collector yang kita buat tadi
	"arm-monitoring-sse/collector"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// ... (setting header SSE & CORS tetap sama) ...
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	for {
		// Panggil fungsi dari package collector dengan prefix 'collector.'
		downSpeed, upSpeed := collector.GetNetworkSpeed()

		status := collector.ServerStatus{
			CPUUsage: collector.GetCPUUsage(),
			CPUTemp:  collector.GetCPUTemperature(),
			Download: downSpeed, // Masukkan data bersih download
			Upload:   upSpeed,   // Masukkan data bersih upload
			// RAMUsage:   collector.GetRAMUsage(),
			// Containers: collector.GetDockerContainers(),
		}

		jsonData, _ := json.Marshal(status)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()

		time.Sleep(1 * time.Second)
	}
}
