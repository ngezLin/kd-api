package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type WhatsAppMessage struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// SendWhatsAppNotification mengirim notifikasi WhatsApp Menggunakan API dari fonnte.com
func SendWhatsAppNotification(phone, message string) error {
	apiURL := "https://api.fonnte.com/send"
	token := os.Getenv("FONNTE_TOKEN") // Ambil dari environment variable

	if token == "" {
		return fmt.Errorf("FONNTE_TOKEN tidak ditemukan di environment")
	}

	payload := map[string]string{
		"target":  phone,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("gagal membuat request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API mengembalikan status: %d", resp.StatusCode)
	}

	return nil
}

// FormatTransactionMessage memformat pesan transaksi
func FormatTransactionMessage(transactionID uint, status string, total float64, items []string) string {
	message := "TRANSAKSI BARU\n\n"
	message += fmt.Sprintf("ID: #%d\n", transactionID)
	message += fmt.Sprintf("Status: %s\n", status)
	message += fmt.Sprintf("Total: Rp %.0f\n\n", total)
	message += "*Items:*\n"
	
	for i, item := range items {
		message += fmt.Sprintf("%d. %s\n", i+1, item)
	}
	
	message += fmt.Sprintf("\n_Waktu: %s_", time.Now().Format("02/01/2006 15:04:05"))
	
	return message
}