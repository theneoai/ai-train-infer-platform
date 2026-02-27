# API Documentation

## Overview

AITIP provides a RESTful API for managing AI training, inference, and datasets.

**Base URL**: `http://localhost:8080/api/v1`

## Authentication

### JWT Token

Most endpoints require JWT authentication. Include the token in the Authorization header:

```http
Authorization: Bearer <your-jwt-token>
```

### API Key

Alternative authentication using API Key:

```http
X-API-Key: <your-api-key>
```

## Endpoints

### Authentication

#### Register
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "username",
  "password": "password"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "username"
    },
    "token": "jwt-token"
  }
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password"
}
```

#### Get Current User
```http
GET /auth/me
Authorization: Bearer <token>
```

### Datasets

#### List Datasets
```http
GET /datasets
Authorization: Bearer <token>
```

**Query Parameters**:
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20)

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "name": "mnist-dataset",
      "format": "zip",
      "size_bytes": 104857600,
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100
  }
}
```

#### Create Dataset (Upload)
```http
POST /datasets
Authorization: Bearer <token>
Content-Type: multipart/form-data

file: <binary-file-data>
name: "dataset-name"
description: "Dataset description"
```

#### Download Dataset
```http
GET /datasets/:id/download
Authorization: Bearer <token>
```

### Training Jobs

#### List Training Jobs
```http
GET /training/jobs
Authorization: Bearer <token>
```

#### Create Training Job
```http
POST /training/jobs
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "mnist-training",
  "dataset_id": "dataset-uuid",
  "config": {
    "framework": "pytorch",
    "model": "resnet18",
    "epochs": 10,
    "batch_size": 32,
    "learning_rate": 0.001
  },
  "gpu_count": 1
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "job-uuid",
    "name": "mnist-training",
    "status": "pending",
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

#### Get Training Job
```http
GET /training/jobs/:id
Authorization: Bearer <token>
```

#### Get Training Logs (SSE Stream)
```http
GET /training/jobs/:id/logs
Authorization: Bearer <token>
Accept: text/event-stream
```

**Response** (Server-Sent Events):
```
data: {"timestamp": "2025-01-15T10:01:00Z", "level": "INFO", "message": "Training started"}

data: {"timestamp": "2025-01-15T10:01:05Z", "level": "INFO", "message": "Epoch 1/10, Loss: 2.123"}
```

#### Stop Training Job
```http
POST /training/jobs/:id/stop
Authorization: Bearer <token>
```

### Inference Services

#### List Services
```http
GET /inference/services
Authorization: Bearer <token>
```

#### Create Service
```http
POST /inference/services
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "mnist-inference",
  "model_id": "model-uuid",
  "runtime": "triton",
  "gpu_count": 1
}
```

#### Start Service
```http
POST /inference/services/:id/start
Authorization: Bearer <token>
```

#### Stop Service
```http
POST /inference/services/:id/stop
Authorization: Bearer <token>
```

## Error Codes

| Code | Status | Description |
|------|--------|-------------|
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource already exists |
| 500 | Internal Server Error | Server error |

## Error Response Format

```json
{
  "success": false,
  "error": {
    "code": "Bad Request",
    "message": "Invalid training configuration"
  }
}
```
