//go:build !linux

package kvm

import (
	"context"
	"errors"
)

type LibvirtManager struct{ address string }

func NewLibvirtManager(ctx context.Context, address string, bridge string) (*LibvirtManager, error) {
	return nil, errors.New("libvirt manager is only supported on linux")
}

func (l *LibvirtManager) CreateVM(ctx context.Context, req CreateVMRequest) (VM, error) {
	return VM{}, errors.New("libvirt manager is only supported on linux")
}
func (l *LibvirtManager) DeleteVM(ctx context.Context, id string) error {
	return errors.New("libvirt manager is only supported on linux")
}
func (l *LibvirtManager) StartVM(ctx context.Context, id string) error {
	return errors.New("libvirt manager is only supported on linux")
}
func (l *LibvirtManager) StopVM(ctx context.Context, id string) error {
	return errors.New("libvirt manager is only supported on linux")
}
func (l *LibvirtManager) GetVM(ctx context.Context, id string) (VM, error) {
	return VM{}, errors.New("libvirt manager is only supported on linux")
}
func (l *LibvirtManager) ListVMs(ctx context.Context) ([]VM, error) {
	return nil, errors.New("libvirt manager is only supported on linux")
}
