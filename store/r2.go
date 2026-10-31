package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Bucket is the name of the R2 bucket of the raw store.
const Bucket = "olymmath-raw"

// R2 is a Remote that uses the Cloudflare API for the objects of an R2 bucket. The API token needs the permission to edit R2 objects.
type R2 struct {
	Account string
	Token   string
	Bucket  string
	Client  *http.Client
	// Base is the address of the API. It is for the tests.
	Base string
}

// R2FromEnv returns an R2 remote from CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN. It returns nil when one of them is empty.
func R2FromEnv() *R2 {
	acct, tok := os.Getenv("CLOUDFLARE_ACCOUNT_ID"), os.Getenv("CLOUDFLARE_API_TOKEN")
	if acct == "" || tok == "" {
		return nil
	}
	b := os.Getenv("OLYMMATH_RAW_BUCKET")
	if b == "" {
		b = Bucket
	}
	return &R2{Account: acct, Token: tok, Bucket: b}
}

func (r *R2) url(key string) string {
	base := r.Base
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return fmt.Sprintf("%s/accounts/%s/r2/buckets/%s/objects/%s", base, r.Account, r.Bucket, strings.Join(parts, "/"))
}

func (r *R2) client() *http.Client {
	if r.Client != nil {
		return r.Client
	}
	return http.DefaultClient
}

func (r *R2) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.Token)
	resp, err := r.client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 == 2 {
		return resp, nil
	}
	defer resp.Body.Close()
	var body struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	_ = json.Unmarshal(b, &body)
	msg := resp.Status
	if len(body.Errors) > 0 {
		msg = fmt.Sprintf("%s: %d %s", resp.Status, body.Errors[0].Code, body.Errors[0].Message)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("R2 %s: %w", req.URL.Path, ErrNotFound)
	}
	return nil, fmt.Errorf("R2 %s %s: %s", req.Method, req.URL.Path, msg)
}

// Put writes the object key.
func (r *R2) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.url(key), body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := r.do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Get reads the object key.
func (r *R2) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url(key), nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Delete removes the object key. It is not an error when the object does not exist.
func (r *R2) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.url(key), nil)
	if err != nil {
		return err
	}
	resp, err := r.do(req)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
