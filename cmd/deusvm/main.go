package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riccardotacconi/deusvm/internal/api"
	"github.com/riccardotacconi/deusvm/internal/config"
	"github.com/riccardotacconi/deusvm/internal/kvm"
	"github.com/riccardotacconi/deusvm/internal/logging"
	"github.com/riccardotacconi/deusvm/internal/storage"
	deusvmproto "github.com/riccardotacconi/deusvm/pkg/proto/gen/github.com/riccardotacconi/deusvm/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := logging.New()
	defer func() { _ = logger.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", logging.FieldError(err))
	}

	var manager kvm.Manager
	// For now, use in-memory manager unless LIBVIRT_ADDR is set or config.Libvirt.Address present
	libvirtAddr := cfg.Libvirt.Address
	if env := os.Getenv("LIBVIRT_ADDR"); env != "" {
		libvirtAddr = env
	}
	if libvirtAddr != "" {
		lm, lerr := kvm.NewLibvirtManager(ctx, libvirtAddr, cfg.Network.Bridge)
		if lerr != nil {
			logger.Warn("failed to connect to libvirt, falling back to in-memory manager", logging.FieldError(lerr))
			manager = kvm.NewInMemoryManager()
		} else {
			manager = lm
		}
	} else {
		manager = kvm.NewInMemoryManager()
	}

	store, err := storage.NewLocalManager(cfg.Storage.ImagesPath)
	if err != nil {
		logger.Fatal("failed to init storage", logging.FieldError(err))
	}

	apiServer := api.NewServer(logger, manager, store, cfg)

	server := &http.Server{
		Addr:              cfg.API.ListenAddress,
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting DeusVM API server", logging.Field("addr", cfg.API.ListenAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", logging.FieldError(err))
		}
	}()

	// Start gRPC server
	go func() {
		lisAddr := cfg.GRPC.ListenAddress
		var opts []grpc.ServerOption
		if cfg.GRPC.TLS.Enabled {
			creds, err := credentials.NewServerTLSFromFile(cfg.GRPC.TLS.CertFile, cfg.GRPC.TLS.KeyFile)
			if err != nil {
				logger.Fatal("failed to load TLS certs", logging.FieldError(err))
			}
			opts = append(opts, grpc.Creds(creds))
		}
		grpcServer := grpc.NewServer(opts...)
		deusvmproto.RegisterVMServiceServer(grpcServer, api.NewVMServiceServer(manager))
		deusvmproto.RegisterImageServiceServer(grpcServer, api.NewImageServiceServer(store))
		ln, err := netListen("tcp", lisAddr)
		if err != nil {
			logger.Fatal("gRPC listen error", logging.FieldError(err))
		}
		logger.Info("starting gRPC server", logging.Field("addr", lisAddr))
		if err := grpcServer.Serve(ln); err != nil {
			logger.Fatal("gRPC server error", logging.FieldError(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "server shutdown error: %v\n", err)
	}
}
