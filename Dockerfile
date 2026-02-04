# ---- Builder Stage ----
# Use a Debian-based slim image which has pre-compiled wheels for OpenCV,
# avoiding the need for a lengthy compilation process.
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.* /app
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o shape-detector ./cmd/main.go

# ---- Final Stage ----
# This stage creates the lean, final image for runtime.
FROM alpine:latest


WORKDIR /app

# Install only the runtime OS dependencies for OpenCV

COPY --from=builder /app/shape-detector /app/shape-detectors

# Expose the port Gunicorn will run on
EXPOSE 8080

# Command to run the application using Gunicorn
CMD ["/app/shape-detectors", "-p", "8080"]