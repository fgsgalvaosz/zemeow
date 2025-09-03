# ZeMeow - WhatsApp Integration API

## Docker Compose Files

This project includes several Docker Compose files for different environments and purposes:

### 1. Development Environment
- `docker-compose.dev.yml` - For local development with all services (PostgreSQL, Redis, MinIO)

### 2. Production Environment
- `docker-compose.yml` - Complete setup with all services for production deployment

### 3. Docker Hub Deployment
- `docker-compose.push.yml` - Minimal setup for building and pushing the application image to Docker Hub

## Building and Pushing to Docker Hub

To build and push the ZeMeow application image to Docker Hub:

1. Login to Docker Hub:
   ```bash
   docker login
   ```

2. Build the image:
   ```bash
   docker compose -f docker-compose.push.yml build
   ```

3. Push the image to Docker Hub:
   ```bash
   docker compose -f docker-compose.push.yml push
   ```

## Environment Variables

Make sure to configure your environment variables in the `.env` file. You can copy the example file to get started:

```bash
cp .env.example .env
```

Then edit the `.env` file with your specific configuration.

## Running the Application

To run the complete application with all services:

```bash
docker compose up -d
```

To run only the development services (without the main application):

```bash
docker compose -f docker-compose.dev.yml up -d
```