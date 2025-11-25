# Video Processing Service

A high-performance, distributed video processing service built with Go, designed to handle video transcoding, HLS streaming, and thumbnail generation at scale.

## Features

- **Multiple Resolution Support**: Automatically processes videos into multiple resolutions (1080p, 720p, 480p, 360p, 240p, 144p)
- **HLS Streaming**: Generates HTTP Live Streaming (HLS) playlists and segments for adaptive bitrate streaming
- **Thumbnail Generation**: Automatically creates thumbnails for each video variant
- **Distributed Processing**: Uses Redis for job queuing and distributed processing
- **Scalable Storage**: Integrates with MinIO for scalable object storage
- **Database Integration**: Stores video metadata in PostgreSQL
- **RESTful API**: Provides endpoints for video upload, status checking, and playback

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Client     │    │  API        │    │  Worker     │
│ (Web/Mobile)│◄──►│  Service    │◄──►│  Processes  │
└─────────────┘    └─────────────┘    └─────────────┘
       ▲                  ▲                  │
       │                  │                  │
       ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Web UI     │    │  Redis      │    │  MinIO      │
│  (Optional) │    │  (streaming)│    │  (storage)  │
└─────────────┘    └─────────────┘    └─────────────┘
                                             │
                                             ▼
                                       ┌─────────────┐
                                       │  PostgreSQL │
                                       │  (Metadata) │
                                       └─────────────┘
```

## Prerequisites

- Docker and Docker Compose
- Go 1.20+
- FFmpeg

## Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/video-processing.git
   cd video-processing
   ```

2. **Set up environment variables**
   Copy the example environment file and update the values:
   ```bash
   cp .env.example .env
   ```

3. **Start the services**
   ```bash
   docker compose up -d
   ```

4. **Run migrations**
   ```bash
   go run cmd/migrate/main.go
   ```

5. **Start the application**
   ```bash
   go run cmd/api/main.go
   ```

## API Documentation

### Interactive API Documentation

The API is documented using Swagger/OpenAPI. After starting the application, you can access the interactive documentation at:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **OpenAPI JSON**: `http://localhost:8080/swagger/doc.json`

### Available Endpoints

- `POST /api/v1/videos` - Upload a new video
- `GET /api/v1/videos` - List all videos
- `GET /api/v1/videos/:id` - Get video details
- `GET /api/v1/videos/:id/stream` - Stream a video
- `DELETE /api/v1/videos/:id` - Delete a video

### Generating API Documentation

API documentation is automatically generated from code comments using [swag](https://github.com/swaggo/swag). To update the documentation:

1. Install swag:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. Regenerate docs:
   ```bash
   swag init -g cmd/api/main.go
   ```

3. Restart the application to apply changes

## Configuration

Environment variables can be set in the `.env` file:

```env
# Server
PORT=8080
ENV=development

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=video_processing

# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# JWT
JWT_SECRET=your_jwt_secret
JWT_EXPIRATION=24h
```

## Development

### Running Tests

```bash
go test -v ./...
```

### Building the Application

```bash
go build -o bin .
```

### Linting

```bash
golangci-lint run
```

## Deployment

The application is designed to be deployed using Docker containers. A `docker-compose.prod.yml` file is provided for production deployment.

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## CI/CD Pipeline

The project includes a GitHub Actions workflow (`.github/workflows/ci-cd.yml`) that runs on every push and pull request to the `main` branch. The pipeline includes:

### Test Job
- Runs on: Push and Pull Requests
- Services:
  - PostgreSQL 15
  - Redis
  - MinIO
- Steps:
  - Linting with golangci-lint
  - Swagger documentation generation
  - Unit and integration tests with coverage
  - Code coverage upload to Codecov

### Build Job (Main Branch Only)
- Runs on: Push to `main`
- Builds and pushes a Docker image to github container registry
- Requires github container registry credentials in repository secrets

### Required Secrets
- `GITHUB_USERNAME`: Your github username
- `GITHUB_TOKEN`: Your github access token

## Monitoring

The application exposes Prometheus metrics at `/metrics` for monitoring. You can use Grafana to visualize the metrics.

## License

This project is no longer licensed.
## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request
