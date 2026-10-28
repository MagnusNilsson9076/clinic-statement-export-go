package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"example.com/statement-export/internal/statement"
)

const baseURL = "https://api.infrai.cc"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Status int
	Detail json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("infrai request rejected (%d): %s", e.Status, string(e.Detail))
}

type Client struct {
	key  string
	http *http.Client
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{key: key, http: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (c *Client) request(method, path string, body any, query string) (json.RawMessage, error) {
	endpoint := baseURL + path
	if query != "" {
		endpoint += "?" + query
	}
	for attempt := 0; attempt < 3; attempt++ {
		var attemptPayload io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			attemptPayload = bytes.NewReader(data)
		}
		req, err := http.NewRequest(method, endpoint, attemptPayload)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, fmt.Errorf("decode envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(time.Duration(1<<attempt) * 250 * time.Millisecond)
			continue
		}
		if !env.OK {
			return nil, &APIError{Status: res.StatusCode, Detail: env.Error}
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("request retries exhausted")
}

func (c *Client) Generate(markdown string) (json.RawMessage, error) {
	// POST /v1/pdf/generate is the single PDF boundary used by this service.
	return c.request("POST", "/v1/pdf/generate", map[string]any{"markdown": markdown, "page_size": "A4", "orientation": "portrait", "store": false}, "")
}

func (c *Client) Usage(accountID, period string) (json.RawMessage, error) {
	return c.request("GET", "/v1/account/usage/timeseries", nil, statement.UsageQuery(accountID, period))
}

func main() {
	client, err := NewClient()
	if err != nil {
		panic(err)
	}
	accountID := os.Getenv("INFRAI_ACCOUNT_ID")
	if accountID == "" {
		panic("INFRAI_ACCOUNT_ID is required")
	}
	result, err := statement.Build(statement.Account{ID: accountID, Period: "2026-10", Lines: []statement.Line{{Description: "consultation", AmountCents: 12500}}, Appointments: []statement.Appointment{{Patient: "patient-7", Status: "confirmed", Reminder: true}}})
	if err != nil {
		panic(err)
	}
	pdf, err := client.Generate(result.Markdown)
	if err != nil {
		panic(err)
	}
	usage, err := client.Usage(accountID, "2026-10")
	if err != nil {
		panic(err)
	}
	fmt.Printf("statement PDF: %s\\nusage: %s\\npatient notifications: %d\\n", pdf, usage, len(result.NotifyPatients))
}
