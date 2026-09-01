.PHONY: build-node build-cli build-gui docker-build test lint clean

build-node:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rookeryd ./node/cmd/rookeryd

# Builds the node's Docker image locally, e.g. for testing before it's built
# by CI and published to ghcr.io on push (see .github/workflows/node-docker.yml).
docker-build:
	docker build -f node/Dockerfile -t rookery-node .

build-cli:
	go build -o bin/rookery-cli ./client/cmd/rookery-cli

# Requires Wails CLI (github.com/wailsapp/wails/v2/cmd/wails) and WebView2
# Runtime. Build on Windows, or in CI on windows-latest. wintun.dll is
# loaded at runtime (not linked), so it has to sit next to the exe.
build-gui:
	cd client/gui && wails build
	cp client/gui/wintun/wintun.dll client/gui/build/bin/wintun.dll

test:
	go test ./client/... ./node/... ./shared/...

lint:
	go vet ./client/... ./node/... ./shared/...

clean:
	rm -rf bin/
