# Production Dockerfile for Driver Service
# Multi-stage build for security and optimization

FROM golang:1.25 AS build

WORKDIR /app 

# Optimize dependency caching by copying only go.mod and go.sum first
# This enables Docker layer caching for dependencies
COPY go.mod go.sum ./

# Cache mounts significantly speed up dependency downloads
# These caches persist across builds, improving build times
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM build AS build-production

# Add non-root user for security
RUN useradd -u 1001 rideshare

# Copy source code
COPY . .

# Build the Driver Service binary
# We compile statically to avoid runtime dependencies in final image
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o driver-service \
    ./services/driver-service

# Final production stage - using scratch for minimal attack surface
FROM scratch

WORKDIR /

# Copy passwd file for non-root user
COPY --from=build-production /etc/passwd /etc/passwd

# Copy the binary (statically linked - includes all dependencies)
COPY --from=build-production /app/driver-service /driver-service

# Use non-root user for security
USER rideshare

# Expose port (adjust as needed)
EXPOSE 8083

# Run the binary
CMD ["/driver-service"]
