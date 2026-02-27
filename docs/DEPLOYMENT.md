# Deployment Guide

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- Git

## Quick Start with Docker Compose

### 1. Clone Repository

```bash
git clone https://github.com/theneoai/ai-train-infer-platform.git
cd ai-train-infer-platform
```

### 2. Start Services

```bash
make dev
```

This will start:
- PostgreSQL (port 5432)
- Redis (port 6379)
- MinIO (port 9000/9001)
- Gateway API (port 8080)
- All backend services
- Web UI (port 3000)

### 3. Run Database Migrations

```bash
make migrate-up
```

### 4. Access Services

- **Web UI**: http://localhost:3000
- **API**: http://localhost:8080
- **MinIO Console**: http://localhost:9001
  - Username: minioadmin
  - Password: minioadmin

## Configuration

### Environment Variables

Create `.env` file in project root:

```env
# Database
DATABASE_URL=postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# JWT
JWT_SECRET_KEY=your-secret-key-change-in-production

# Services
USER_SERVICE_URL=http://user:8081
DATA_SERVICE_URL=http://data:8082
TRAINING_SERVICE_URL=http://training:8083
INFERENCE_SERVICE_URL=http://inference:8084
```

### GPU Support

For NVIDIA GPU support, ensure:

1. NVIDIA Docker runtime is installed:
```bash
# Install nvidia-docker2
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list

sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo systemctl restart docker
```

2. Add GPU support to docker-compose.yml:
```yaml
training:
  # ... other config
  deploy:
    resources:
      reservations:
        devices:
          - driver: nvidia
            count: all
            capabilities: [gpu]
```

## Production Deployment

### Kubernetes

See `deploy/k8s/` directory for Kubernetes manifests.

```bash
kubectl apply -k deploy/k8s/overlays/prod/
```

### Helm

```bash
cd deploy/helm/aitip
helm install aitip . -f values-prod.yaml
```

## Monitoring

### View Logs

```bash
# All services
make logs

# Specific service
docker-compose -f deploy/docker-compose.yml logs -f training
```

### Health Checks

```bash
curl http://localhost:8080/health
```

## Troubleshooting

### Database Connection Failed

Check PostgreSQL is running:
```bash
docker-compose -f deploy/docker-compose.yml ps postgres
```

### MinIO Connection Failed

Verify MinIO credentials in environment variables.

### GPU Not Available

Check NVIDIA driver and nvidia-docker installation:
```bash
nvidia-smi
docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi
```

## Upgrade

```bash
# Pull latest code
git pull origin main

# Rebuild and restart
make dev-stop
make dev
make migrate-up
```
