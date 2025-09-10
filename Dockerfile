FROM golang:1.24.4-alpine AS builder

WORKDIR /src

# Install git for version info
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build-time metadata
ARG VERSION=unknown
ARG COMMIT=unknown
ARG DATE=unknown

# Build the binary with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${DATE}'" -o /app/mcp-digitalocean ./cmd/mcp-digitalocean

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/mcp-digitalocean ./mcp-digitalocean

# Expose default port
EXPOSE 8080

# Entrypoint allows passing all supported flags
ENTRYPOINT ["/app/mcp-digitalocean"]
CMD ["--help"]

