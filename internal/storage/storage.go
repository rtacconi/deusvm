package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Image struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	Format string `json:"format"`
	SHA256 string `json:"sha256"`
}

type Manager interface {
	SaveImageFromURL(ctx context.Context, name, sourceURL string) (Image, error)
	ListImages(ctx context.Context) ([]Image, error)
	DeleteImage(ctx context.Context, name string) error
	CreateDiskFromBase(ctx context.Context, baseImageName, diskName string, sizeBytes int64) (string, error)
}

type LocalManager struct {
	imagesDir string
}

func NewLocalManager(imagesDir string) (*LocalManager, error) {
	if imagesDir == "" {
		return nil, errors.New("imagesDir required")
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir images: %w", err)
	}
	return &LocalManager{imagesDir: imagesDir}, nil
}

func (m *LocalManager) imagePath(name string) (string, error) {
	if name == "" || strings.Contains(name, "..") || strings.ContainsRune(name, filepath.Separator) {
		return "", errors.New("invalid image name")
	}
	return filepath.Join(m.imagesDir, name), nil
}

func (m *LocalManager) SaveImageFromURL(ctx context.Context, name, sourceURL string) (Image, error) {
	path, err := m.imagePath(name)
	if err != nil {
		return Image{}, err
	}
	// stream download to file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return Image{}, fmt.Errorf("new request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Image{}, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Image{}, fmt.Errorf("download status: %s", resp.Status)
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return Image{}, fmt.Errorf("create tmp: %w", err)
	}
	defer f.Close()
	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	if err != nil {
		_ = os.Remove(tmp)
		return Image{}, fmt.Errorf("write: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = os.Remove(tmp)
		return Image{}, fmt.Errorf("sync: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return Image{}, fmt.Errorf("rename: %w", err)
	}
	img := Image{
		Name:   name,
		Path:   path,
		Size:   n,
		Format: detectFormatByName(name),
		SHA256: fmt.Sprintf("%x", hasher.Sum(nil)),
	}
	return img, nil
}

func (m *LocalManager) ListImages(ctx context.Context) ([]Image, error) {
	entries, err := os.ReadDir(m.imagesDir)
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}
	var out []Image
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		path := filepath.Join(m.imagesDir, name)
		out = append(out, Image{Name: name, Path: path, Size: info.Size(), Format: detectFormatByName(name)})
	}
	return out, nil
}

func (m *LocalManager) DeleteImage(ctx context.Context, name string) error {
	path, err := m.imagePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func detectFormatByName(name string) string {
	low := strings.ToLower(name)
	switch {
	case strings.HasSuffix(low, ".qcow2"):
		return "qcow2"
	case strings.HasSuffix(low, ".raw"):
		return "raw"
	default:
		return "unknown"
	}
}

// CreateDiskFromBase copies the base image to a new disk path (placeholder implementation).
func (m *LocalManager) CreateDiskFromBase(ctx context.Context, baseImageName, diskName string, sizeBytes int64) (string, error) {
	base, err := m.imagePath(baseImageName)
	if err != nil {
		return "", err
	}
	out, err := m.imagePath(diskName)
	if err != nil {
		return "", err
	}
	src, err := os.Open(base)
	if err != nil {
		return "", fmt.Errorf("open base: %w", err)
	}
	defer src.Close()
	dst, err := os.Create(out)
	if err != nil {
		return "", fmt.Errorf("create disk: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(out)
		return "", fmt.Errorf("copy: %w", err)
	}
	if err := dst.Sync(); err != nil {
		_ = os.Remove(out)
		return "", fmt.Errorf("sync: %w", err)
	}
	return out, nil
}
