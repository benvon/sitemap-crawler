# Dockerfile for GoReleaser
# GoReleaser will build the binary and copy it into this image
FROM alpine:latest

# Install ca-certificates for HTTPS requests and wget for debugging
RUN apk --no-cache add ca-certificates wget

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# GoReleaser will copy the binary with the project name
# Copy additional files
COPY README.md .
COPY LICENSE .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Run the application
# The binary name will match the project name (sitemap-crawler)
CMD ["./sitemap-crawler"]
