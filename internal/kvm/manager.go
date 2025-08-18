package kvm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type VMStatus string

const (
	VMStatusUnknown VMStatus = "unknown"
	VMStatusStopped VMStatus = "stopped"
	VMStatusRunning VMStatus = "running"
)

type VM struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CPU         int       `json:"cpu"`
	MemoryBytes int64     `json:"memory_bytes"`
	DiskBytes   int64     `json:"disk_bytes"`
	Image       string    `json:"image"`
	Status      VMStatus  `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateVMRequest struct {
	Name        string
	CPU         int
	MemoryBytes int64
	DiskBytes   int64
	Image       string
}

type Manager interface {
	CreateVM(ctx context.Context, req CreateVMRequest) (VM, error)
	DeleteVM(ctx context.Context, id string) error
	StartVM(ctx context.Context, id string) error
	StopVM(ctx context.Context, id string) error
	GetVM(ctx context.Context, id string) (VM, error)
	ListVMs(ctx context.Context) ([]VM, error)
}

// InMemoryManager is a functional placeholder used for local development and API plumbing tests.
type InMemoryManager struct {
	mu      sync.RWMutex
	vms     map[string]VM
	nameIdx map[string]string
}

func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{vms: make(map[string]VM), nameIdx: make(map[string]string)}
}

func (m *InMemoryManager) CreateVM(ctx context.Context, req CreateVMRequest) (VM, error) {
	if req.Name == "" || req.CPU <= 0 || req.MemoryBytes <= 0 || req.DiskBytes <= 0 {
		return VM{}, fmt.Errorf("invalid create request")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.nameIdx[req.Name]; exists {
		return VM{}, fmt.Errorf("vm with name %q already exists", req.Name)
	}
	id := uuid.NewString()
	vm := VM{
		ID:          id,
		Name:        req.Name,
		CPU:         req.CPU,
		MemoryBytes: req.MemoryBytes,
		DiskBytes:   req.DiskBytes,
		Image:       req.Image,
		Status:      VMStatusStopped,
		CreatedAt:   time.Now().UTC(),
	}
	m.vms[id] = vm
	m.nameIdx[vm.Name] = id
	return vm, nil
}

func (m *InMemoryManager) DeleteVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vm, ok := m.vms[id]
	if !ok {
		return notFound(id)
	}
	delete(m.vms, id)
	delete(m.nameIdx, vm.Name)
	return nil
}

func (m *InMemoryManager) StartVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vm, ok := m.vms[id]
	if !ok {
		return notFound(id)
	}
	vm.Status = VMStatusRunning
	m.vms[id] = vm
	return nil
}

func (m *InMemoryManager) StopVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vm, ok := m.vms[id]
	if !ok {
		return notFound(id)
	}
	vm.Status = VMStatusStopped
	m.vms[id] = vm
	return nil
}

func (m *InMemoryManager) GetVM(ctx context.Context, id string) (VM, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vm, ok := m.vms[id]
	if !ok {
		return VM{}, notFound(id)
	}
	return vm, nil
}

func (m *InMemoryManager) ListVMs(ctx context.Context) ([]VM, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]VM, 0, len(m.vms))
	for _, vm := range m.vms {
		list = append(list, vm)
	}
	return list, nil
}

func notFound(id string) error { return fmt.Errorf("vm %s not found", id) }
