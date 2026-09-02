# Build context is the repo root (needs ../shared alongside panel/), e.g.:
#   docker build -f panel/Dockerfile -t rookery-panel .
FROM golang:1.27-alpine AS builder

ENV GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

COPY shared/ shared/
COPY panel/ panel/

WORKDIR /src/panel
RUN go build -o /out/rookeryp ./cmd/rookeryp

# Plain (root) distroless, not :nonroot — /data is a bind/named volume the
# operator owns, and there's no shell here to chown it to a fixed non-root
# UID at container start. Distroless still has no shell/package manager, so
# running as root doesn't add much attack surface.
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/rookeryp /rookeryp
ENTRYPOINT ["/rookeryp", "-config", "/etc/rookery/panel.yaml"]
