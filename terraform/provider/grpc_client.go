package provider

import (
	"context"

	deusvmproto "github.com/riccardotacconi/deusvm/pkg/proto/gen/github.com/riccardotacconi/deusvm/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClients struct {
	VM    deusvmproto.VMServiceClient
	Image deusvmproto.ImageServiceClient
	conn  *grpc.ClientConn
}

func NewGRPCClients(ctx context.Context, endpoint string, useTLS bool) (*GRPCClients, error) {
	var opts []grpc.DialOption
	if useTLS {
		// TODO: plumb TLS config for provider
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}
	return &GRPCClients{
		VM:    deusvmproto.NewVMServiceClient(conn),
		Image: deusvmproto.NewImageServiceClient(conn),
		conn:  conn,
	}, nil
}

func (c *GRPCClients) Close() error { return c.conn.Close() }
