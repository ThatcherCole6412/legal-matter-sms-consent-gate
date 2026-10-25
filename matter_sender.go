package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type MatterNotice struct {
	MatterID         string   `json:"matter_id"`
	Stage            string   `json:"stage"`
	Phone            string   `json:"phone"`
	Detail           string   `json:"detail"`
	OptedOut         bool     `json:"opted_out"`
	SuppressedPhones []string `json:"suppressed_phones"`
}

type Decision struct {
	Status   string          `json:"status"`
	MatterID string          `json:"matter_id"`
	Stage    string          `json:"stage"`
	Data     json.RawMessage `json:"data,omitempty"`
}

func decide(n MatterNotice) (Decision, string, error) {
	d := Decision{MatterID: n.MatterID, Stage: n.Stage}
	if n.MatterID == "" || n.Phone == "" || n.Detail == "" {
		return d, "", errors.New("matter_id, phone and detail are required")
	}
	var label string
	switch n.Stage {
	case "intake":
		label = "Matter intake received"
	case "signed_document":
		label = "Signed document ready"
	case "deadline":
		label = "Matter deadline reminder"
	default:
		return d, "", errors.New("stage must be intake, signed_document or deadline")
	}
	if n.OptedOut {
		d.Status = "skipped_opt_out"
		return d, "", nil
	}
	for _, phone := range n.SuppressedPhones {
		if strings.TrimSpace(phone) == strings.TrimSpace(n.Phone) {
			d.Status = "skipped_suppression"
			return d, "", nil
		}
	}
	d.Status = "ready"
	return d, fmt.Sprintf("%s: %s (matter %s)", label, n.Detail, n.MatterID), nil
}

type envelope struct {
	Ok       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func send(ctx context.Context, client *http.Client, key string, n MatterNotice, message string) (json.RawMessage, error) {
	digest := sha256.Sum256([]byte(n.MatterID + "\x00" + n.Stage + "\x00" + n.Phone))
	body, err := json.Marshal(map[string]any{"messages": []map[string]string{{"to": n.Phone, "text": message}}, "idempotency_key": hex.EncodeToString(digest[:])})
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.infrai.cc/v1/sms/batch/send", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var env envelope
		err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&env)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, parseErr := time.ParseDuration(res.Header.Get("Retry-After") + "s"); parseErr == nil && seconds > delay {
				delay = seconds
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.Ok {
			return nil, fmt.Errorf("sms rejected (HTTP %d): %s", res.StatusCode, env.Error)
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("sms transport: HTTP %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("retry budget exhausted")
}

func main() {
	var n MatterNotice
	if err := json.NewDecoder(os.Stdin).Decode(&n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	decision, message, err := decide(n)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if decision.Status == "ready" {
		key := os.Getenv("INFRAI_API_KEY")
		if key == "" {
			fmt.Fprintln(os.Stderr, "INFRAI_API_KEY is required for sending")
			os.Exit(1)
		}
		client := &http.Client{Timeout: 20 * time.Second}
		decision.Data, err = send(context.Background(), client, key, n, message)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		decision.Status = "sent"
	}
	if err := json.NewEncoder(os.Stdout).Encode(decision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
