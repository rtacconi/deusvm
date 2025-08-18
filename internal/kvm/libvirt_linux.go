//go:build linux

package kvm

import (
	"context"
	"fmt"
	"strings"
	"time"

	libvirt "libvirt.org/go/libvirt"
)

// LibvirtManager implements Manager using libvirt on Linux.
type LibvirtManager struct {
	address string
}

func NewLibvirtManager(ctx context.Context, address string) (*LibvirtManager, error) {
	if address == "" {
		address = "qemu:///system"
	}
	// Defer full connection until operations to avoid failing fast on startup.
	return &LibvirtManager{address: address}, nil
}

func (l *LibvirtManager) dial() (*libvirt.Connect, error) {
	conn, err := libvirt.NewConnect(l.address)
	if err != nil {
		return nil, fmt.Errorf("libvirt connect: %w", err)
	}
	return conn, nil
}

func (l *LibvirtManager) CreateVM(ctx context.Context, req CreateVMRequest) (VM, error) {
	if req.Name == "" || req.CPU <= 0 || req.MemoryBytes <= 0 || req.Image == "" {
		return VM{}, fmt.Errorf("invalid create request")
	}
	conn, err := l.dial()
	if err != nil {
		return VM{}, err
	}
	defer conn.Close()

	memoryKiB := req.MemoryBytes / 1024
	diskType := "raw"
	low := strings.ToLower(req.Image)
	if strings.HasSuffix(low, ".qcow2") {
		diskType = "qcow2"
	}

	domainXML := fmt.Sprintf(`
<domain type='kvm'>
  <name>%s</name>
  <memory unit='KiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='%s'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <graphics type='vnc' autoport='yes'/>
  </devices>
</domain>`, req.Name, memoryKiB, req.CPU, diskType, req.Image)

	dom, err := conn.DomainDefineXML(domainXML)
	if err != nil {
		return VM{}, fmt.Errorf("define domain: %w", err)
	}
	defer dom.Free()
	uuidStr, _ := dom.GetUUIDString()
	vm := VM{
		ID:          uuidStr,
		Name:        req.Name,
		CPU:         req.CPU,
		MemoryBytes: req.MemoryBytes,
		DiskBytes:   req.DiskBytes,
		Image:       req.Image,
		Status:      VMStatusStopped,
		CreatedAt:   time.Now().UTC(),
	}
	return vm, nil
}

func (l *LibvirtManager) DeleteVM(ctx context.Context, id string) error {
	conn, err := l.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	dom, err := conn.LookupDomainByUUIDString(id)
	if err != nil {
		dom, err = conn.LookupDomainByName(id)
	}
	if err != nil {
		return fmt.Errorf("lookup domain: %w", err)
	}
	defer dom.Free()
	active, _ := dom.IsActive()
	if active {
		_ = dom.Destroy()
	}
	if err := dom.Undefine(); err != nil {
		return fmt.Errorf("undefine: %w", err)
	}
	return nil
}

func (l *LibvirtManager) StartVM(ctx context.Context, id string) error {
	conn, err := l.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	dom, err := conn.LookupDomainByUUIDString(id)
	if err != nil {
		// try by name fallback
		dom, err = conn.LookupDomainByName(id)
	}
	if err != nil {
		return fmt.Errorf("lookup domain: %w", err)
	}
	defer dom.Free()
	if err := dom.Create(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	return nil
}

func (l *LibvirtManager) StopVM(ctx context.Context, id string) error {
	conn, err := l.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	dom, err := conn.LookupDomainByUUIDString(id)
	if err != nil {
		dom, err = conn.LookupDomainByName(id)
	}
	if err != nil {
		return fmt.Errorf("lookup domain: %w", err)
	}
	defer dom.Free()
	if err := dom.Shutdown(); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func (l *LibvirtManager) GetVM(ctx context.Context, id string) (VM, error) {
	conn, err := l.dial()
	if err != nil {
		return VM{}, err
	}
	defer conn.Close()
	dom, err := conn.LookupDomainByUUIDString(id)
	if err != nil {
		dom, err = conn.LookupDomainByName(id)
	}
	if err != nil {
		return VM{}, fmt.Errorf("lookup domain: %w", err)
	}
	defer dom.Free()
	name, _ := dom.GetName()
	info, err := dom.GetInfo()
	if err != nil {
		return VM{}, fmt.Errorf("get info: %w", err)
	}
	status := VMStatusStopped
	if info.State == libvirt.DOMAIN_RUNNING {
		status = VMStatusRunning
	}
	uuidStr, _ := dom.GetUUIDString()
	vm := VM{
		ID:          uuidStr,
		Name:        name,
		CPU:         int(info.NrVirtCpu),
		MemoryBytes: int64(info.Memory) * 1024,
		Status:      status,
	}
	return vm, nil
}

func (l *LibvirtManager) ListVMs(ctx context.Context) ([]VM, error) {
	conn, err := l.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	doms, err := conn.ListAllDomains(0)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	var out []VM
	for _, d := range doms {
		name, _ := d.GetName()
		info, _ := d.GetInfo()
		uuidStr, _ := d.GetUUIDString()
		status := VMStatusStopped
		if info != nil && info.State == libvirt.DOMAIN_RUNNING {
			status = VMStatusRunning
		}
		out = append(out, VM{ID: uuidStr, Name: name, CPU: int(info.NrVirtCpu), MemoryBytes: int64(info.Memory) * 1024, Status: status})
		d.Free()
	}
	return out, nil
}
