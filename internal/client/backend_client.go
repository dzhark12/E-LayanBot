package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"elayan.local/desa/internal/storage"
)

var ErrNotFound = errors.New("not found")

type Backend struct {
	baseURL string
	token   string
	files   *storage.Local
	http    *http.Client
}

type Attachment struct {
	Key  string
	Kind string
	MIME string
	Size int64
}

type Submission struct {
	RequestID      string
	LetterCode     string
	WAJID          string
	Summary        string
	ConsentVersion string
	Attachments    []Attachment
}

type SubmissionResult struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

type LetterType struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Requirements []string `json:"requirements"`
}

type StatusResult struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

func NewBackend(baseURL, token string, files *storage.Local) (*Backend, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("invalid bot API URL")
	}
	return &Backend{baseURL: strings.TrimRight(baseURL, "/"), token: token, files: files, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (b *Backend) LetterTypes(ctx context.Context) ([]LetterType, error) {
	var result []LetterType
	err := b.getJSON(ctx, "/api/v1/letter-types", &result)
	return result, err
}

func (b *Backend) FAQ(ctx context.Context, query string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/api/v1/faq?q="+url.QueryEscape(query), nil)
	if err != nil {
		return "", err
	}
	resp, err := b.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAQ API returned %d", resp.StatusCode)
	}
	var result struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return "", err
	}
	return result.Answer, nil
}

func (b *Backend) CreateSubmission(ctx context.Context, input Submission) (SubmissionResult, error) {
	if len(input.Attachments) == 0 || len(input.Attachments) > 5 {
		return SubmissionResult{}, errors.New("invalid attachment count")
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, field := range [][2]string{
		{"letter_code", input.LetterCode},
		{"request_id", input.RequestID},
		{"wa_jid", input.WAJID},
		{"summary", input.Summary},
		{"consent_version", input.ConsentVersion},
	} {
		if err := form.WriteField(field[0], field[1]); err != nil {
			return SubmissionResult{}, err
		}
	}
	for _, attachment := range input.Attachments {
		if attachment.Size < 1 || attachment.Size > 10<<20 || attachment.MIME != "image/jpeg" && attachment.MIME != "image/png" || attachment.Kind != "KTP" && attachment.Kind != "KK" && attachment.Kind != "RT" && attachment.Kind != "LAINNYA" {
			return SubmissionResult{}, errors.New("invalid attachment metadata")
		}
		file, err := b.files.Open(attachment.Key)
		if err != nil {
			return SubmissionResult{}, err
		}
		filename := "lampiran.jpg"
		if attachment.MIME == "image/png" {
			filename = "lampiran.png"
		}
		part, partErr := form.CreateFormFile("attachments", filename)
		if partErr == nil {
			var written int64
			written, partErr = io.Copy(part, io.LimitReader(file, (10<<20)+1))
			if partErr == nil && (written < 1 || written > 10<<20 || written != attachment.Size) {
				partErr = errors.New("invalid staged attachment size")
			}
		}
		closeErr := file.Close()
		if partErr != nil {
			return SubmissionResult{}, partErr
		}
		if closeErr != nil {
			return SubmissionResult{}, closeErr
		}
		if err := form.WriteField("attachment_types", attachment.Kind); err != nil {
			return SubmissionResult{}, err
		}
		if body.Len() > 52<<20 {
			return SubmissionResult{}, errors.New("submission exceeds size limit")
		}
	}
	if err := form.Close(); err != nil {
		return SubmissionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/api/v1/submissions", &body)
	if err != nil {
		return SubmissionResult{}, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := b.do(req)
	if err != nil {
		return SubmissionResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return SubmissionResult{}, fmt.Errorf("submission API returned %d", resp.StatusCode)
	}
	var result SubmissionResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return SubmissionResult{}, err
	}
	return result, nil
}

func (b *Backend) CheckStatus(ctx context.Context, code, waJID string) (StatusResult, error) {
	body, err := json.Marshal(map[string]string{"code": code, "waJid": waJID})
	if err != nil {
		return StatusResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/api/v1/status", bytes.NewReader(body))
	if err != nil {
		return StatusResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.do(req)
	if err != nil {
		return StatusResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return StatusResult{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return StatusResult{}, fmt.Errorf("status API returned %d", resp.StatusCode)
	}
	var result StatusResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return StatusResult{}, err
	}
	return result, nil
}

func (b *Backend) getJSON(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := b.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(target)
}

func (b *Backend) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.http.Do(req)
}
