package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
)

func rawPut(baseURL, apiKey string, id int32, body map[string]interface{}) {
	b, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/sub-account/%d", baseURL, id)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  err: %v\n", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("  body=%s -> HTTP %d %s\n", string(b), resp.StatusCode, string(respBody))
}

func main() {
	cfg, _ := config.Load()
	client := mailtarget.NewSubAccountClient(cfg.Mailtarget.APIConfigURL, cfg.Mailtarget.APIKey)
	sa, err := client.Get(context.Background(), 4302)
	if err != nil {
		panic(err)
	}
	fmt.Printf("current status: %s name=%q ipPoolId=%d\n\n", sa.Status, sa.SubAccountName, sa.IPPoolID)

	if sa.Status != mailtarget.StatusActive {
		if _, err := client.Resume(context.Background(), 4302); err != nil {
			fmt.Println("resume failed:", err)
		}
		sa, err = client.Get(context.Background(), 4302)
		if err != nil {
			panic(err)
		}
		fmt.Printf("after resume: %s\n\n", sa.Status)
	}

	tests := []map[string]interface{}{
		{"name": sa.SubAccountName, "status": "Suspended", "ipPoolId": sa.IPPoolID},
		{"name": sa.SubAccountName, "status": "Suspended"},
		{"status": "Suspended"},
		{"name": sa.SubAccountName, "status": "suspended", "ipPoolId": sa.IPPoolID},
	}
	for i, t := range tests {
		fmt.Printf("test %d:\n", i+1)
		rawPut(cfg.Mailtarget.APIConfigURL, cfg.Mailtarget.APIKey, 4302, t)
		// restore Active between tests
		_, _ = client.Resume(context.Background(), 4302)
	}
}
