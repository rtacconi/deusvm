package api

import (
	"context"

	"github.com/riccardotacconi/deusvm/internal/kvm"
	"github.com/riccardotacconi/deusvm/internal/storage"
	deusvmproto "github.com/riccardotacconi/deusvm/pkg/proto/gen/github.com/riccardotacconi/deusvm/pkg/proto"
)

type VMServiceServer struct {
	deusvmproto.UnimplementedVMServiceServer
	manager kvm.Manager
}

func NewVMServiceServer(manager kvm.Manager) *VMServiceServer {
	return &VMServiceServer{manager: manager}
}

func (s *VMServiceServer) Create(ctx context.Context, req *deusvmproto.CreateVMRequest) (*deusvmproto.VM, error) {
	vm, err := s.manager.CreateVM(ctx, kvm.CreateVMRequest{
		Name: req.GetName(), Image: req.GetImage(), CPU: int(req.GetCpu()), MemoryBytes: req.GetMemoryBytes(), DiskBytes: req.GetDiskBytes(),
	})
	if err != nil {
		return nil, err
	}
	return vmToProto(vm), nil
}

func (s *VMServiceServer) Delete(ctx context.Context, req *deusvmproto.VMIDRequest) (*deusvmproto.Empty, error) {
	if err := s.manager.DeleteVM(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &deusvmproto.Empty{}, nil
}

func (s *VMServiceServer) Start(ctx context.Context, req *deusvmproto.VMIDRequest) (*deusvmproto.Empty, error) {
	if err := s.manager.StartVM(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &deusvmproto.Empty{}, nil
}

func (s *VMServiceServer) Stop(ctx context.Context, req *deusvmproto.VMIDRequest) (*deusvmproto.Empty, error) {
	if err := s.manager.StopVM(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &deusvmproto.Empty{}, nil
}

func (s *VMServiceServer) Get(ctx context.Context, req *deusvmproto.VMIDRequest) (*deusvmproto.VM, error) {
	vm, err := s.manager.GetVM(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return vmToProto(vm), nil
}

func (s *VMServiceServer) List(ctx context.Context, req *deusvmproto.Empty) (*deusvmproto.ListVMsResponse, error) {
	vms, err := s.manager.ListVMs(ctx)
	if err != nil {
		return nil, err
	}
	out := &deusvmproto.ListVMsResponse{}
	for _, vm := range vms {
		out.Vms = append(out.Vms, vmToProto(vm))
	}
	return out, nil
}

func vmToProto(vm kvm.VM) *deusvmproto.VM {
	return &deusvmproto.VM{
		Id:          vm.ID,
		Name:        vm.Name,
		Cpu:         int32(vm.CPU),
		MemoryBytes: vm.MemoryBytes,
		DiskBytes:   vm.DiskBytes,
		Image:       vm.Image,
		Status:      string(vm.Status),
	}
}

type ImageServiceServer struct {
	deusvmproto.UnimplementedImageServiceServer
	storage storage.Manager
}

func NewImageServiceServer(store storage.Manager) *ImageServiceServer {
	return &ImageServiceServer{storage: store}
}

func (s *ImageServiceServer) Create(ctx context.Context, req *deusvmproto.CreateImageRequest) (*deusvmproto.Image, error) {
	img, err := s.storage.SaveImageFromURL(ctx, req.GetName(), req.GetSource())
	if err != nil {
		return nil, err
	}
	return &deusvmproto.Image{Name: img.Name, Path: img.Path, SizeBytes: img.Size, Format: img.Format, Sha256: img.SHA256}, nil
}

func (s *ImageServiceServer) Delete(ctx context.Context, req *deusvmproto.ImageNameRequest) (*deusvmproto.Empty, error) {
	if err := s.storage.DeleteImage(ctx, req.GetName()); err != nil {
		return nil, err
	}
	return &deusvmproto.Empty{}, nil
}

func (s *ImageServiceServer) List(ctx context.Context, req *deusvmproto.Empty) (*deusvmproto.ListImagesResponse, error) {
	imgs, err := s.storage.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	out := &deusvmproto.ListImagesResponse{}
	for _, im := range imgs {
		out.Images = append(out.Images, &deusvmproto.Image{Name: im.Name, Path: im.Path, SizeBytes: im.Size, Format: im.Format, Sha256: im.SHA256})
	}
	return out, nil
}
