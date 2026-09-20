# TightShip — one binary, one command to build it.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -s -w -X main.version=$(VERSION)

.PHONY: all build test vet web run check clean

all: web build

build:            ## static binary with whatever frontend build is in internal/webui/dist
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/tightship ./cmd/tightship

test:
	go test ./...

vet:
	go vet ./...

web:              ## build the web app into internal/webui/dist (needs node)
	cd web && npm ci && npm run build

run: build        ## run against the example config with the debug identity header allowed
	TIGHTSHIP_DB_PASSWORD=x ./bin/tightship serve --config deploy/config.dev.yaml

check: build
	./bin/tightship check --config deploy/config.example.yaml

clean:
	rm -rf bin internal/webui/dist/* web/node_modules
	touch internal/webui/dist/.gitkeep
