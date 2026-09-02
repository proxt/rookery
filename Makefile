.PHONY: build-panel build-cli build-gui docker-build test lint clean

build-panel:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rookeryp ./panel/cmd/rookeryp

# Builds the panel's Docker image locally, e.g. for testing before it's built
# by CI and published to ghcr.io on push (see .github/workflows/panel-docker.yml).
docker-build:
	docker build -f panel/Dockerfile -t rookery-panel .

build-cli:
	go build -o bin/rookery-cli ./client/cmd/rookery-cli

# Requires Wails CLI (github.com/wailsapp/wails/v2/cmd/wails) and WebView2
# Runtime. Build on Windows, or in CI on windows-latest. wintun.dll is
# loaded at runtime (not linked), so it has to sit next to the exe.
build-gui:
	cd client/gui && wails build
	cp client/gui/wintun/wintun.dll client/gui/build/bin/wintun.dll

test:
	go test ./client/... ./panel/... ./shared/...

lint:
	go vet ./client/... ./panel/... ./shared/...

clean:
	rm -rf bin/
