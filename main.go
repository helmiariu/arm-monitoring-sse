package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Daftarkan handler dari file handler.go
	http.HandleFunc("/stream-status", statusHandler)

	fmt.Println("Server SSE berjalan di port :4646")
	http.ListenAndServe(":4646", nil)
}
