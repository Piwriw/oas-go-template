# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

# Allow overriding GOPROXY for builds in restricted networks (e.g. behind GFW).
# Default uses the standard Go module proxy.
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a static binary
# CGO_DISABLED=1 + -ldflags '-extldflags "-static"' for static link
# trimpath removes absolute paths for reproducibility
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w -extldflags "-static"' \
    -o /out/server \
    ./cmd/server

# Runtime stage
FROM alpine:3.20

# ca-certificates for HTTPS calls, tzdata for time zone support
RUN apk --no-cache add ca-certificates tzdata

# Non-root user
RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=builder /out/server /app/server

ENV HTTP_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
