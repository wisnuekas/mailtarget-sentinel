package mailtarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TransmissionClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewTransmissionClient(baseURL, apiKey string) *TransmissionClient {
	return &TransmissionClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TransmissionClient) Send(ctx context.Context, form TransmissionForm) (*TransmissionData, error) {
	body, err := json.Marshal(form)
	if err != nil {
		return nil, fmt.Errorf("marshal transmission form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/layang/transmissions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send transmission: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("transmission API %d: %s", resp.StatusCode, firstNonEmpty(errResp.Message, errResp.Error, string(respBody)))
	}

	var data TransmissionData
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &data, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return "unknown error"
}
