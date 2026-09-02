# Build context is the repo root (needs ../shared alongside panel/), e.g.:
#   docker build -f panel/Dockerfile -t rookery-panel .
FROM node:20-alpine AS frontend
WORKDIR /src/panel/internal/admin/frontend
COPY panel/internal/admin/frontend/package.json panel/internal/admin/frontend/package-lock.json* ./
RUN npm install
COPY panel/internal/admin/frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS builder

ARG BUILD_TIME=unknown
ARG BUILD_COMMIT=unknown

ENV GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

COPY shared/ shared/
COPY panel/ panel/
# go:embed needs the built frontend present before `go build` runs.
COPY --from=frontend /src/panel/internal/admin/frontend/dist panel/internal/admin/frontend/dist

WORKDIR /src/panel
RUN go build -ldflags "-X github.com/rookery/panel/internal/buildinfo.BuildTime=${BUILD_TIME} -X github.com/rookery/panel/internal/buildinfo.Commit=${BUILD_COMMIT}" \
    -o /out/rookeryp ./cmd/rookeryp

# Plain (root) distroless, not :nonroot — /data is a bind/named volume the
# operator owns, and there's no shell here to chown it to a fixed non-root
# UID at container start. Distroless still has no shell/package manager, so
# running as root doesn't add much attack surface.
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/rookeryp /rookeryp
ENTRYPOINT ["/rookeryp", "-config", "/etc/rookery/panel.yaml"]
