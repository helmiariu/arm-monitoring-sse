package collector

type ContainerStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ServerStatus struct {
	CPUUsage float64 `json:"cpu"`
	CPUTemp  float64 `json:"temp"`
	Download float64 `json:"download"` // Satuan KB/s
	Upload   float64 `json:"upload"`   // Satuan KB/s
	// RAMUsage   float64           `json:"ram"`
	// Uptime     string            `json:"uptime"`
	// DownloadKB float64           `json:"download_kb"`
	// UploadKB   float64           `json:"upload_kb"`
	// Containers []ContainerStatus `json:"containers"`
}
