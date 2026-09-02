// Package buildinfo holds build-time metadata injected via -ldflags, so the
// admin UI can show when the running image was actually built — the only
// reliable way to confirm an auto-update (via Watchtower) really landed.
package buildinfo

var (
	// BuildTime is the UTC build timestamp (RFC3339), set by the Docker
	// build. Left as "unknown" for local `go build` / `go run`.
	BuildTime = "unknown"
	// Commit is the short git SHA the image was built from.
	Commit = "unknown"
)
