# DeusVM – Golang KVM Hypervisor Manager

A single-server KVM manager written in Go with native Terraform integration. DeusVM provides a modern, DevOps-friendly alternative to Proxmox for single-node use cases, exposing a first-party gRPC API (protobuf) and a REST API for third-party consumers.

## Highlights

- First-party interface: gRPC (Protocol Buffers)
- Third-party interface: REST (HTTP/JSON)
- VM lifecycle: create, start, stop, list, delete
- Image management: upload (by URL), list, delete
- Terraform provider (plugin framework v1)
- Linux-only libvirt integration (with macOS/Windows stubs for development builds)
- CLI (`deusvmctl`) using gRPC only
- Structured logging with zap

## Architecture

- `cmd/deusvm`: Daemon process
  - Starts REST server (default `:8080`) for third parties
  - Starts gRPC server (default `:9090`) for first-party tooling and Terraform provider
  - Wraps managers:
    - `internal/kvm` – VM lifecycle (in-memory for dev; libvirt-backed on Linux)
    - `internal/storage` – Local filesystem image/disk management
- `cmd/deusvmctl`: CLI that talks to DeusVM via gRPC only (protobuf) and provides VM and image subcommands.
- `terraform/provider`: Terraform provider using the plugin framework, talking gRPC to the daemon
- `pkg/proto`: Protocol Buffer definitions and generated Go stubs
- REST API is implemented with chi for convenience and for third-party consumers

### Data plane choices

- KVM/libvirt integration: `libvirt.org/go/libvirt` (CGo) on Linux, non-Linux builds provide stubs
- Storage: local filesystem under `/var/lib/deusvm/{images,disks}`
- Networking: bridged networking intended; domain XML currently includes disk and VNC; NIC/bridge wiring is designed for extension

## Repository layout

```
cmd/
  deusvm/               # Main daemon: starts REST and gRPC services
  deusvmctl/            # CLI using gRPC (protobuf) only
internal/
  api/                  # REST handlers and gRPC service implementations
  config/               # YAML/env configuration loader (Viper)
  kvm/                  # KVM/libvirt manager (linux impl + non-linux stubs), in-memory impl for dev
  logging/              # zap logger helpers
  storage/              # Image and disk management on local FS
pkg/
  proto/
    deusvm.proto        # Protobuf definitions
    gen/...             # Generated Go code (do not edit)
  client/               # Minimal REST client (useful for third-party tools)
terraform/
  provider/             # Terraform provider implementation (gRPC-based)
scripts/
  debian-systemd/       # Systemd unit for Debian/Ubuntu
Dockerfile              # Distroless runtime image for the daemon
README.md               # This file
```

## Makefile

Common tasks are provided via a Makefile:

- Format: `make fmt`
- Tidy modules: `make tidy`
- Build binaries (to `./bin`): `make build`
- Generate protobufs: `make proto` (run `make proto-tools` once to install plugins)
- Run daemon: `make run`
- Build Docker image: `make docker-build`

## Configuration

DeusVM reads config from `/etc/deusvm/deusvm.yaml` (or `./deusvm.yaml`) and environment variables (prefix `DEUSVM_`).

Supported settings:

- `api.listen_address`: REST listen address (default `:8080`)
- `api.auth_token`: optional Bearer token to protect REST endpoints
- `grpc.listen_address`: gRPC listen address (default `:9090`)
- `grpc.tls.enabled`: enable TLS on gRPC (default `false`)
- `grpc.tls.cert_file`: path to TLS cert (PEM)
- `grpc.tls.key_file`: path to TLS key (PEM)
- `storage.images_path`: path for images (default `/var/lib/deusvm/images`)
- `storage.disks_path`: path for VM disks (default `/var/lib/deusvm/disks`)
- `network.bridge`: Linux bridge name (default `br0`)
- `libvirt.address`: libvirt URI (e.g., `qemu:///system`)

Environment variable overrides example: `DEUSVM_API_LISTEN_ADDRESS=":8081"`.

### Example `/etc/deusvm/deusvm.yaml`

```yaml
api:
  listen_address: ":8080"
  auth_token: ""

grpc:
  listen_address: ":9090"
  tls:
    enabled: false
    cert_file: "/etc/deusvm/tls/server.crt"
    key_file: "/etc/deusvm/tls/server.key"

storage:
  images_path: "/var/lib/deusvm/images"
  disks_path: "/var/lib/deusvm/disks"

network:
  bridge: "br0"

libvirt:
  address: "qemu:///system"
```

## Build (local)

Prerequisites:
- Go ≥ 1.22
- For protobuf changes only: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc` (not required to run/build if generated code is committed)

Commands:

- Build daemon: `go build ./cmd/deusvm` or `make build`
- Build CLI: `go build ./cmd/deusvmctl` (created by `make build`)
- Build Terraform provider: `go build ./cmd/terraform-provider-deusvm` (created by `make build`)

Run locally (dev mode):
- Start daemon: `./bin/deusvm` (or `./deusvm` if built directly)
- CLI examples:
  - `./bin/deusvmctl image create --name debian-13.qcow2 --source https://.../debian-13.qcow2`
  - `./bin/deusvmctl vm create --name web-01 --image /var/lib/deusvm/images/debian-13.qcow2 --cpu 2 --memory 4GB --disk 20GB`
  - `./bin/deusvmctl vm list`

## gRPC and REST

- gRPC (protobuf): primary API for first-party tools (CLI, Terraform). See `pkg/proto/deusvm.proto`.
- REST: secondary API for 3rd-party users/integrations. Available at `/api/v1/...`.

## Terraform provider (dev)

The provider uses gRPC to talk to the daemon.

Example usage (skeleton):

```hcl
terraform {
  required_providers {
    deusvm = {
      source  = "deusvm/deusvm"
      version = ">= 0.0.1"
    }
  }
}

provider "deusvm" {
  endpoint = "127.0.0.1:9090"
}

resource "deusvm_image" "debian" {
  name   = "debian-13.qcow2"
  source = "https://cloud.debian.org/images/cloud/trixie/daily/.../debian-13.qcow2"
}

resource "deusvm_vm" "web" {
  name   = "web-01"
  image  = "/var/lib/deusvm/images/debian-13.qcow2"
  cpu    = 2
  memory = "4GB"
  disk   = "20GB"
}
```

Notes:
- For local development, you can set up a dev provider override. See Terraform docs on `CLI configuration file` and `provider_installation` to point to your built binary.

## Deployment on Debian (tutorial)

These steps assume a Debian/Ubuntu server with virtualization support.

### 1) Install KVM/libvirt

```bash
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils
sudo systemctl enable --now libvirtd
# Verify
lsmod | grep kvm
virsh list --all
```

### 2) Configure a bridge (br0)

Configure your network so the host uses a bridge `br0` and the VM NICs attach to it. One basic approach on Debian using `/etc/network/interfaces`:

```bash
sudo tee /etc/network/interfaces.d/br0 <<'EOF'
auto br0
iface br0 inet dhcp
    bridge_ports eno1
    bridge_stp off
    bridge_fd 0
    bridge_maxwait 0
EOF

sudo systemctl restart networking || sudo reboot
```

Adjust `bridge_ports` (physical NIC) and addressing model to your environment. On systems with Netplan/systemd-networkd/NetworkManager, configure an equivalent bridge there.

### 3) Create DeusVM directories

```bash
sudo mkdir -p /etc/deusvm /var/lib/deusvm/{images,disks} /etc/deusvm/tls
sudo chown -R root:root /var/lib/deusvm
```

### 4) Install the DeusVM daemon

Option A: Copy a locally built binary

```bash
# On your dev machine
GOOS=linux GOARCH=amd64 go build -o deusvm ./cmd/deusvm
scp deusvm user@your-server:/tmp/deusvm

# On the server
sudo mv /tmp/deusvm /usr/local/bin/deusvm
sudo chmod +x /usr/local/bin/deusvm
```

Option B: Build on the server (requires Go 1.22+)

```bash
sudo apt install -y golang git
sudo mkdir -p /opt/deusvm && sudo chown "$USER" /opt/deusvm
cd /opt/deusvm
git clone https://github.com/riccardotacconi/deusvm .
go build ./cmd/deusvm
sudo cp deusvm /usr/local/bin/
```

### 5) Configure DeusVM

Create `/etc/deusvm/deusvm.yaml` (see example above). Minimal example:

```yaml
api:
  listen_address: ":8080"

grpc:
  listen_address: ":9090"

storage:
  images_path: "/var/lib/deusvm/images"
  disks_path: "/var/lib/deusvm/disks"

network:
  bridge: "br0"

libvirt:
  address: "qemu:///system"
```

Optional: place TLS keypair under `/etc/deusvm/tls/` and set `grpc.tls.enabled: true` with paths.

### 6) Install systemd service

```bash
sudo cp /opt/deusvm/scripts/debian-systemd/deusvm.service /etc/systemd/system/deusvm.service
# or copy from your working directory accordingly

sudo systemctl daemon-reload
sudo systemctl enable --now deusvm
sudo systemctl status deusvm --no-pager
```

### 7) Open firewall (if applicable)

- gRPC: TCP 9090
- REST: TCP 8080 (optional)

### 8) Test with CLI from your workstation

```bash
# Build CLI locally (or on server and scp back)
go build ./cmd/deusvmctl

# Test connectivity
./deusvmctl image list --endpoint your-server:9090

# Add an image
./deusvmctl image create --endpoint your-server:9090 \
  --name debian-13.qcow2 \
  --source https://cloud.debian.org/images/cloud/trixie/daily/.../debian-13.qcow2

# Create a VM
./deusvmctl vm create --endpoint your-server:9090 \
  --name web-01 --image /var/lib/deusvm/images/debian-13.qcow2 \
  --cpu 2 --memory 4GB --disk 20GB

# List VMs
./deusvmctl vm list --endpoint your-server:9090
```

## Development notes

- Libvirt integration is Linux-only; on non-Linux hosts, the project builds with stubs so REST/gRPC and in-memory manager can still be exercised.
- The domain XML currently sets up a disk and VNC display; you can extend it to attach a bridged NIC using `network.bridge`.
- The Terraform provider currently demonstrates create/delete flows. Reads and updates will evolve with the API.

## Security notes

- Restrict REST to trusted networks or protect with `api.auth_token`.
- Prefer enabling gRPC TLS for production.
- Ensure directories under `/var/lib/deusvm` have appropriate permissions.

## License

Apache-2.0 (pending) – adjust as needed.
