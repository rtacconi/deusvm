package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	deusvmproto "github.com/riccardotacconi/deusvm/pkg/proto/gen/github.com/riccardotacconi/deusvm/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "vm":
		vmCmd(os.Args[2:])
	case "image":
		imageCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(1)
	}
}

func dials(endpoint string) (*grpc.ClientConn, deusvmproto.VMServiceClient, deusvmproto.ImageServiceClient, error) {
	if endpoint == "" {
		endpoint = "127.0.0.1:9090"
	}
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, nil, err
	}
	return conn, deusvmproto.NewVMServiceClient(conn), deusvmproto.NewImageServiceClient(conn), nil
}

func vmCmd(args []string) {
	if len(args) == 0 {
		vmUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("vm create", flag.ExitOnError)
		var endpoint, name, image, memory, disk string
		var cpu int
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		fs.StringVar(&name, "name", "", "VM name")
		fs.StringVar(&image, "image", "", "base image path or name")
		fs.IntVar(&cpu, "cpu", 1, "vCPU count")
		fs.StringVar(&memory, "memory", "1GB", "memory (e.g. 4GB)")
		fs.StringVar(&disk, "disk", "10GB", "disk size (e.g. 20GB)")
		_ = fs.Parse(args[1:])
		if name == "" || image == "" {
			fmt.Fprintln(os.Stderr, "name and image required")
			os.Exit(1)
		}
		memBytes, err := parseSize(memory)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid memory")
			os.Exit(1)
		}
		diskBytes, err := parseSize(disk)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid disk")
			os.Exit(1)
		}
		conn, vmc, _, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		vm, err := vmc.Create(ctx, &deusvmproto.CreateVMRequest{Name: name, Image: image, Cpu: int32(cpu), MemoryBytes: memBytes, DiskBytes: diskBytes})
		if err != nil {
			fatal(err)
		}
		fmt.Println(vm.GetId())
	case "list":
		fs := flag.NewFlagSet("vm list", flag.ExitOnError)
		var endpoint string
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		_ = fs.Parse(args[1:])
		conn, vmc, _, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resp, err := vmc.List(ctx, &deusvmproto.Empty{})
		if err != nil {
			fatal(err)
		}
		for _, v := range resp.GetVms() {
			fmt.Printf("%s\t%s\t%s\n", v.GetId(), v.GetName(), v.GetStatus())
		}
	case "get":
		fs := flag.NewFlagSet("vm get", flag.ExitOnError)
		var endpoint, id string
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		fs.StringVar(&id, "id", "", "VM id or name")
		_ = fs.Parse(args[1:])
		if id == "" {
			fmt.Fprintln(os.Stderr, "id required")
			os.Exit(1)
		}
		conn, vmc, _, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		v, err := vmc.Get(ctx, &deusvmproto.VMIDRequest{Id: id})
		if err != nil {
			fatal(err)
		}
		fmt.Printf("%s\t%s\t%d CPU\t%d MB\t%s\n", v.GetId(), v.GetName(), v.GetCpu(), v.GetMemoryBytes()/1024/1024, v.GetStatus())
	case "delete":
		vmAction(args[1:], "vm delete", func(ctx context.Context, vmc deusvmproto.VMServiceClient, id string) error {
			_, err := vmc.Delete(ctx, &deusvmproto.VMIDRequest{Id: id})
			return err
		})
	case "start":
		vmAction(args[1:], "vm start", func(ctx context.Context, vmc deusvmproto.VMServiceClient, id string) error {
			_, err := vmc.Start(ctx, &deusvmproto.VMIDRequest{Id: id})
			return err
		})
	case "stop":
		vmAction(args[1:], "vm stop", func(ctx context.Context, vmc deusvmproto.VMServiceClient, id string) error {
			_, err := vmc.Stop(ctx, &deusvmproto.VMIDRequest{Id: id})
			return err
		})
	default:
		vmUsage()
		os.Exit(1)
	}
}

func vmAction(args []string, name string, fn func(context.Context, deusvmproto.VMServiceClient, string) error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	var endpoint, id string
	fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
	fs.StringVar(&id, "id", "", "VM id or name")
	_ = fs.Parse(args)
	if id == "" {
		fmt.Fprintln(os.Stderr, "id required")
		os.Exit(1)
	}
	conn, vmc, _, err := dials(endpoint)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := fn(ctx, vmc, id); err != nil {
		fatal(err)
	}
}

func imageCmd(args []string) {
	if len(args) == 0 {
		imageUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("image create", flag.ExitOnError)
		var endpoint, name, source string
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		fs.StringVar(&name, "name", "", "image name (filename)")
		fs.StringVar(&source, "source", "", "source URL")
		_ = fs.Parse(args[1:])
		if name == "" || source == "" {
			fmt.Fprintln(os.Stderr, "name and source required")
			os.Exit(1)
		}
		conn, _, imgc, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := imgc.Create(ctx, &deusvmproto.CreateImageRequest{Name: name, Source: source}); err != nil {
			fatal(err)
		}
		fmt.Println("ok")
	case "list":
		fs := flag.NewFlagSet("image list", flag.ExitOnError)
		var endpoint string
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		_ = fs.Parse(args[1:])
		conn, _, imgc, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := imgc.List(ctx, &deusvmproto.Empty{})
		if err != nil {
			fatal(err)
		}
		for _, im := range resp.GetImages() {
			fmt.Printf("%s\t%s\t%d\n", im.GetName(), im.GetFormat(), im.GetSizeBytes())
		}
	case "delete":
		fs := flag.NewFlagSet("image delete", flag.ExitOnError)
		var endpoint, name string
		fs.StringVar(&endpoint, "endpoint", "127.0.0.1:9090", "gRPC endpoint host:port")
		fs.StringVar(&name, "name", "", "image name")
		_ = fs.Parse(args[1:])
		if name == "" {
			fmt.Fprintln(os.Stderr, "name required")
			os.Exit(1)
		}
		conn, _, imgc, err := dials(endpoint)
		if err != nil {
			fatal(err)
		}
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := imgc.Delete(ctx, &deusvmproto.ImageNameRequest{Name: name}); err != nil {
			fatal(err)
		}
		fmt.Println("ok")
	default:
		imageUsage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("deusvmctl <vm|image> [subcommand] [flags]")
	fmt.Println("Use --help under each subcommand")
}

func vmUsage()    { fmt.Println("vm subcommands: create|list|get|delete|start|stop") }
func imageUsage() { fmt.Println("image subcommands: create|list|delete") }

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return 0, fmt.Errorf("invalid size")
	}
	unit := strings.ToUpper(s[len(s)-2:])
	numStr := s[:len(s)-2]
	var v int64
	for i := 0; i < len(numStr); i++ {
		if numStr[i] < '0' || numStr[i] > '9' {
			return 0, fmt.Errorf("invalid size")
		}
		v = v*10 + int64(numStr[i]-'0')
	}
	switch unit {
	case "GB":
		return v * 1024 * 1024 * 1024, nil
	case "MB":
		return v * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("invalid unit")
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err.Error()); os.Exit(1) }
