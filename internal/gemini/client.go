// Package gemini membungkus Gemini API untuk menjawab pertanyaan warga
// dengan batasan ketat: hanya berdasarkan pengetahuan desa yang diberikan.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://generativelanguage.googleapis.com/v1beta/models/"

var (
	ErrDisabled    = errors.New("gemini disabled")
	ErrRateLimited = errors.New("gemini quota exhausted")
	ErrBlocked     = errors.New("gemini blocked or empty response")
)

// Client memanggil generateContent. Kunci API dikirim lewat header,
// bukan query string, supaya tidak bocor ke log URL/error.
type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: endpoint,
		http:    &http.Client{Timeout: 25 * time.Second},
	}
}

func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" && c.model != "" }

type part struct {
	Text string `json:"text"`
}
type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type request struct {
	SystemInstruction *content         `json:"systemInstruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
	SafetySettings    []safetySetting  `json:"safetySettings,omitempty"`
}
type generationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}
type safetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type response struct {
	Candidates []struct {
		Content      content `json:"content"`
		FinishReason string  `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Generate mengirim instruksi sistem dan pertanyaan terpisah. Memisahkan
// keduanya (systemInstruction vs contents) mempersulit prompt injection
// dibanding menyambung satu string panjang.
func (c *Client) Generate(ctx context.Context, system, user string, maxTokens int) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}
	body, err := json.Marshal(request{
		SystemInstruction: &content{Parts: []part{{Text: system}}},
		Contents:          []content{{Role: "user", Parts: []part{{Text: user}}}},
		GenerationConfig:  generationConfig{MaxOutputTokens: maxTokens},
		SafetySettings: []safetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.model+":generateContent", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("gemini read: %w", err)
	}

	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("gemini decode (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests || out.Error.Status == "RESOURCE_EXHAUSTED" {
		return "", ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK || out.Error.Message != "" {
		// Jangan log isi pertanyaan warga; cukup kode & status dari Google.
		return "", fmt.Errorf("gemini API error: HTTP %d %s", resp.StatusCode, out.Error.Status)
	}
	if out.PromptFeedback.BlockReason != "" || len(out.Candidates) == 0 {
		return "", ErrBlocked
	}
	cand := out.Candidates[0]
	if cand.FinishReason == "SAFETY" || cand.FinishReason == "RECITATION" {
		return "", ErrBlocked
	}
	var sb strings.Builder
	for _, p := range cand.Content.Parts {
		sb.WriteString(p.Text)
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", ErrBlocked
	}
	return text, nil
}
