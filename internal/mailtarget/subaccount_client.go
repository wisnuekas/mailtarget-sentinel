package mailtarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type SubAccountClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewSubAccountClient(baseURL, apiKey string) *SubAccountClient {
	return &SubAccountClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ListSubAccountsParams struct {
	Page      int
	Size      int
	Search    string
	Status    string
	HasAPIKey *bool
}

func (c *SubAccountClient) List(ctx context.Context, p ListSubAccountsParams) (*SubAccountListResponse, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Size <= 0 {
		p.Size = 20
	}

	q := url.Values{}
	q.Set("page", strconv.Itoa(p.Page))
	q.Set("size", strconv.Itoa(p.Size))
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.HasAPIKey != nil {
		q.Set("hasApiKey", strconv.FormatBool(*p.HasAPIKey))
	}

	reqURL := fmt.Sprintf("%s/sub-account?%s", c.baseURL, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("list sub-accounts API %d: %s", resp.StatusCode, firstNonEmpty(errResp.Message, errResp.Error, string(respBody)))
	}

	var data SubAccountListResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &data, nil
}

func (c *SubAccountClient) Get(ctx context.Context, id int32) (*GetSubAccountResponse, error) {
	url := fmt.Sprintf("%s/sub-account/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get sub-account: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("get sub-account API %d: %s", resp.StatusCode, firstNonEmpty(errResp.Message, errResp.Error, string(respBody)))
	}

	var data GetSubAccountResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &data, nil
}

func (c *SubAccountClient) Update(ctx context.Context, id int32, form UpdateSubAccountForm) (*GetSubAccountResponse, error) {
	body, err := json.Marshal(form)
	if err != nil {
		return nil, fmt.Errorf("marshal update form: %w", err)
	}

	url := fmt.Sprintf("%s/sub-account/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update sub-account: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("update sub-account API %d: %s", resp.StatusCode, firstNonEmpty(errResp.Message, errResp.Error, string(respBody)))
	}

	var data GetSubAccountResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &data, nil
}

func (c *SubAccountClient) SetStatus(ctx context.Context, id int32, status string) (*GetSubAccountResponse, error) {
	current, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	ipPoolID := current.IPPoolID
	if ipPoolID == 0 {
		ipPoolID = 1
	}

	return c.Update(ctx, id, UpdateSubAccountForm{
		Name:     current.SubAccountName,
		Status:   status,
		IPPoolID: ipPoolID,
	})
}

func (c *SubAccountClient) Suspend(ctx context.Context, id int32) (*GetSubAccountResponse, error) {
	return c.SetStatus(ctx, id, StatusSuspended)
}

func (c *SubAccountClient) Resume(ctx context.Context, id int32) (*GetSubAccountResponse, error) {
	return c.SetStatus(ctx, id, StatusActive)
}
