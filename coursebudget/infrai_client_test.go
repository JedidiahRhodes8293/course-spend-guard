package coursebudget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletionDecodesOpenAIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"done"}}]}`))
	}))
	defer server.Close()

	client := &Client{baseURL: server.URL, key: "test", httpClient: server.Client()}
	var response struct {
		ID string `json:"id"`
	}
	if err := client.ChatCompletion(context.Background(), "prompt", &response); err != nil {
		t.Fatal(err)
	}
	if response.ID != "chatcmpl-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestChatCompletionPreservesOpenAIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid model","type":"invalid_request_error","code":"model_invalid"}}`))
	}))
	defer server.Close()

	client := &Client{baseURL: server.URL, key: "test", httpClient: server.Client()}
	err := client.ChatCompletion(context.Background(), "prompt", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "model_invalid" || apiErr.Message != "invalid model" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
