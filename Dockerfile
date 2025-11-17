# Dockerfile

# ---- Build Stage ----
# Use an official Go image as the builder
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application as a static binary
# This is crucial for a minimal 'scratch' or 'alpine' final image
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o my-app .

# ---- Final Stage ----
# Use a minimal base image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary and config files from the builder stage
COPY --from=builder /app/my-app .
COPY --from=builder /app/config ./config
COPY --from=builder /app/database/schema ./database/schema

# Expose the port your Go app listens on (e.g., 8080)
EXPOSE 8888

# The command to run your application
CMD ["./my-app"]