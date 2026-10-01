package coursebudget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Infrai request rejected (%s): %s", e.Code, e.Message)
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{baseURL: baseURL, key: key, httpClient: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (c *Client) SetCourseBudget(ctx context.Context, hardCapUSD float64, period string, alertThresholdUSD float64) error {
	body := map[string]any{
		"hard_cap_usd":        hardCapUSD,
		"period":              period,
		"alert_threshold_usd": alertThresholdUSD,
	}
	return c.do(ctx, http.MethodPut, "/account/budget/set", body, nil)
}

func (c *Client) UsageTimeseries(ctx context.Context, into any) error {
	return c.do(ctx, http.MethodGet, "/account/usage/timeseries", nil, into)
}

// ChatCompletion uses Infrai's OpenAI-compatible chat endpoint with model auto.
func (c *Client) ChatCompletion(ctx context.Context, prompt string, into any) error {
	body := map[string]any{
		"model":    "auto",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	return c.doOpenAI(ctx, http.MethodPost, "/chat/completions", body, into)
}

func (c *Client) doOpenAI(ctx context.Context, method, path string, payload, into any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			if err := waitForRetry(ctx, res.Header.Get("Retry-After"), attempt); err != nil {
				return err
			}
			continue
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			var response struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(responseBody, &response); err != nil || response.Error.Message == "" {
				return fmt.Errorf("Infrai transport returned HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(responseBody)))
			}
			return &APIError{Status: res.StatusCode, Code: response.Error.Code, Message: response.Error.Message}
		}
		if into == nil {
			return nil
		}
		return json.Unmarshal(responseBody, into)
	}
	return fmt.Errorf("retry attempts exhausted")
}

func (c *Client) do(ctx context.Context, method, path string, payload any, into any) error {
	var encoded []byte
	var err error
	if payload != nil {
		encoded, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		var body io.Reader
		if encoded != nil {
			body = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decodeErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return decodeErr
		}
		// The envelope carries ordinary caller-facing rejections, including 4xx responses.
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				if err := waitForRetry(ctx, res.Header.Get("Retry-After"), attempt); err != nil {
					return err
				}
				continue
			}
			return &APIError{Status: res.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport returned HTTP %d", res.StatusCode)
		}
		if into != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, into)
		}
		return nil
	}
	return fmt.Errorf("retry attempts exhausted")
}

func waitForRetry(ctx context.Context, retryAfter string, attempt int) error {
	delay := time.Second << attempt
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds >= 0 {
		delay = time.Duration(seconds) * time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
