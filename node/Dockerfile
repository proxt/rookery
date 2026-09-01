# Build context is the repo root (needs ../shared alongside node/), e.g.:
#   docker build -f node/Dockerfile -t rookery-node .
FROM golang:1.27-alpine AS builder

ENV GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

COPY shared/ shared/
COPY node/ node/

WORKDIR /src/node
RUN go build -o /out/rookeryd ./cmd/rookeryd

# Plain (root) distroless, not :nonroot — /data is a bind/named volume the
# operator owns, and there's no shell here to chown it to a fixed non-root
# UID at container start. Distroless still has no shell/package manager, so
# running as root doesn't add much attack surface.
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/rookeryd /rookeryd
ENTRYPOINT ["/rookeryd", "-config", "/etc/rookery/node.yaml"]
