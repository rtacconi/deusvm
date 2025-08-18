package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
}

func New(endpoint, token string) (*Client, error) {
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:    u,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) do(ctx context.Context, method, p string, in any, out any) error {
	u := *c.baseURL
	u.Path = path.Join(u.Path, p)
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Image APIs
type Image struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	Format string `json:"format"`
	SHA256 string `json:"sha256"`
}

func (c *Client) CreateImage(ctx context.Context, name, source string) (Image, error) {
	var out Image
	err := c.do(ctx, http.MethodPost, "/api/v1/images", map[string]string{"name": name, "source": source}, &out)
	return out, err
}

func (c *Client) DeleteImage(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/images/"+name, nil, nil)
}

// VM APIs
type VM struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CPU         int    `json:"cpu"`
	MemoryBytes int64  `json:"memory_bytes"`
	DiskBytes   int64  `json:"disk_bytes"`
	Image       string `json:"image"`
	Status      string `json:"status"`
}

func (c *Client) CreateVM(ctx context.Context, name, image string, cpu int, memory, disk string) (VM, error) {
	var out VM
	payload := map[string]any{
		"name": name, "image": image, "cpu": cpu, "memory": memory, "disk": disk,
	}
	err := c.do(ctx, http.MethodPost, "/api/v1/vms", payload, &out)
	return out, err
}

func (c *Client) GetVM(ctx context.Context, id string) (VM, error) {
	var out VM
	err := c.do(ctx, http.MethodGet, "/api/v1/vms/"+id, nil, &out)
	return out, err
}

func (c *Client) DeleteVM(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/vms/"+id, nil, nil)
}
