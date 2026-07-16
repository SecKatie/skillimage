.PHONY: build install test lint fmt clean image deploy

BINARY      := skillctl
BINDIR      := bin
IMAGE       := ghcr.io/redhat-et/skillctl:latest
INSTALL_DIR ?= /usr/local/bin
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS     ?= -X main.version=$(VERSION)

build:
	mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BINARY) ./cmd/skillctl

install: build
	@echo "Installing $(BINARY) $(VERSION) to $(INSTALL_DIR)/$(BINARY)..."
	@if [ -w "$(INSTALL_DIR)" ]; then \
		install -m 0755 "$(BINDIR)/$(BINARY)" "$(INSTALL_DIR)/$(BINARY)"; \
	else \
		sudo install -m 0755 "$(BINDIR)/$(BINARY)" "$(INSTALL_DIR)/$(BINARY)"; \
	fi
	@echo "Installed $(BINARY) $(VERSION) to $(INSTALL_DIR)/$(BINARY)"
	@"$(INSTALL_DIR)/$(BINARY)" --version

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofumpt -l -w .

image:
	podman -c rhel build -f Dockerfile.local -t $(IMAGE) .

deploy: image
	podman -c rhel push $(IMAGE)
	oc rollout restart deploy/skillctl-catalog
	oc rollout status deploy/skillctl-catalog --timeout=60s
	@echo "---"
	@echo "Route: https://$$(oc get route skillctl-catalog -o jsonpath='{.spec.host}')/api/v1/skills"
	@echo "Run 'oc logs -f deploy/skillctl-catalog' to tail logs"

clean:
	rm -rf $(BINDIR)
