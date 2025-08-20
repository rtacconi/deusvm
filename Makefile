SHELL := /bin/bash

BIN_DIR := bin
PKG := github.com/riccardotacconi/deusvm
PROTO_DIR := pkg/proto
PROTO_OUT := pkg/proto/gen
PROTO_FILES := $(PROTO_DIR)/deusvm.proto

.PHONY: help fmt tidy build clean proto proto-tools run docker-build deps

help:
	@echo "Targets:"
	@echo "  fmt            - go fmt ./..."
	@echo "  tidy           - go mod tidy"
	@echo "  build          - build binaries into ./bin (deusvm, deusvmctl, terraform-provider-deusvm)"
	@echo "  proto-tools    - install protoc-gen-go and protoc-gen-go-grpc"
	@echo "  proto          - generate protobuf stubs into $(PROTO_OUT)"
	@echo "  run            - run deusvm locally"
	@echo "  docker-build   - build docker image deusvm:latest"
	@echo "  deps           - install Debian/Ubuntu server dependencies (apt)"
	@echo "  clean          - remove ./bin"

fmt:
	go fmt ./...

tidy:
	go mod tidy

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: fmt tidy $(BIN_DIR)
	GOFLAGS= go build -o $(BIN_DIR)/deusvm ./cmd/deusvm
	GOFLAGS= go build -o $(BIN_DIR)/deusvmctl ./cmd/deusvmctl
	GOFLAGS= go build -o $(BIN_DIR)/terraform-provider-deusvm ./cmd/terraform-provider-deusvm
	@echo "Binaries built in $(BIN_DIR)"

proto-tools:
	GOFLAGS= go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	GOFLAGS= go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto:
	@which protoc >/dev/null || (echo "protoc not found. Install protobuf compiler." && exit 1)
	@export PATH="$$(go env GOPATH)/bin:$$PATH"; \
	protoc -I $(PROTO_DIR) \
		--go_out=$(PROTO_OUT) \
		--go-grpc_out=$(PROTO_OUT) \
		$(PROTO_FILES)
	@echo "Protobuf generated into $(PROTO_OUT)"

run: build
	$(BIN_DIR)/deusvm

docker-build:
	docker build -t deusvm:latest .

deps:
	@which apt-get >/dev/null || (echo "apt-get not found. This target is for Debian/Ubuntu." && exit 1)
	sudo apt-get update
	sudo apt-get install -y \
		qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils \
		golang make git protobuf-compiler
	@echo "Debian dependencies installed. Consider rebooting or relogging to ensure KVM modules are active."

clean:
	rm -rf $(BIN_DIR)


