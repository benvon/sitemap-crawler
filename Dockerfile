# Dockerfile for GoReleaser
# GoReleaser automatically copies the binary into the build context
FROM ubuntu:latest

# Build argument for binary name (passed by GoReleaser)
ARG BINARY_NAME=sitemap-crawler

# Install ca-certificates for HTTPS requests
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m appuser

WORKDIR /app

# Copy the binary (GoReleaser makes it available in build context)
COPY ${BINARY_NAME} ./${BINARY_NAME}

# Copy additional files (made available via extra_files)
COPY README.md ./README.md
COPY LICENSE ./LICENSE

# Make binary executable and set ownership
RUN chmod +x ./${BINARY_NAME} && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Run the application
CMD ["sh", "-c", "./${BINARY_NAME}"]
