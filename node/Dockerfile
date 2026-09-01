# Build context is the repo root (needs ../shared alongside node/), e.g.:
#   docker build -f node/Dockerfile -t rookery-node .
FROM golang:1.27-alpine AS builder

ENV GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src

COPY shared/ shared/
COPY node/ node/

WORKDIR /src/node
RUN go build -o /out/rookeryd ./cmd/rookeryd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/rookeryd /rookeryd
ENTRYPOINT ["/rookeryd", "-config", "/etc/rookery/node.yaml"]
