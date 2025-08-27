# Dockerfile for GoReleaser
# GoReleaser will build the binary and copy it into this image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the binary (GoReleaser makes this available in the build context)
COPY sitemap-crawler ./sitemap-crawler

# Copy additional files (made available via extra_files)
COPY README.md ./README.md
COPY LICENSE ./LICENSE

# Make binary executable and change ownership to non-root user
RUN chmod +x ./sitemap-crawler && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Run the application
CMD ["./sitemap-crawler"]
