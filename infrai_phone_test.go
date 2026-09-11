package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyCaptchaIncludesWidgetRecordID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["widget_record_id"]; got != "widget-123" {
			t.Errorf("widget_record_id = %v, want widget-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
	}))
	defer server.Close()

	client := NewInfraiClient("test-key")
	client.BaseURL = server.URL
	if err := client.VerifyCaptcha(context.Background(), "widget-123", "captcha-token", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}
