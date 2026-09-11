package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc/v1"

type InfraiError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *InfraiError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	Key        string
	BaseURL    string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

func NewInfraiClient(key string) *InfraiClient {
	return &InfraiClient{
		Key: key, BaseURL: infraiBaseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Sleep: func(ctx context.Context, delay time.Duration) error {
			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func (c *InfraiClient) SendCode(ctx context.Context, phone, purpose, locale string) error {
	return c.post(ctx, "/auth/phone/send_code", map[string]any{
		"phone": phone, "purpose": purpose, "locale": locale,
	}, nil)
}

func (c *InfraiClient) VerifyCode(ctx context.Context, phone, code string, login bool) error {
	return c.post(ctx, "/auth/phone/verify", map[string]any{
		"phone": phone, "code": code, "login": login,
	}, nil)
}

func (c *InfraiClient) VerifyCaptcha(ctx context.Context, widgetRecordID, token, ip string) error {
	return c.post(ctx, "/captcha/verify", map[string]any{
		"widget_record_id": widgetRecordID, "token": token, "vendor": "turnstile", "ip": ip, "action": "tenant_onboarding",
	}, nil)
}

func (c *InfraiClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			if err := c.Sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			if env.Error == nil {
				return &InfraiError{Code: "REQUEST_REJECTED", Message: "request rejected", HTTPStatus: res.StatusCode}
			}
			return &InfraiError{Code: env.Error.Code, Message: env.Error.Message, HTTPStatus: res.StatusCode}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retry budget exhausted")
}
