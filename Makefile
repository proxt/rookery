.PHONY: build-node build-cli build-gui test lint clean

build-node:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rookeryd ./node/cmd/rookeryd

build-cli:
	go build -o bin/rookery-cli ./client/cmd/rookery-cli

# Requires Wails CLI (github.com/wailsapp/wails/v2/cmd/wails) and WebView2
# Runtime. Build on Windows, or in CI on windows-latest.
build-gui:
	cd client/gui && wails build

test:
	go test ./client/... ./node/... ./shared/...

lint:
	go vet ./client/... ./node/... ./shared/...

clean:
	rm -rf bin/
