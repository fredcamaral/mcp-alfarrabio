# Docker Setup Guide

This project provides multiple Docker Compose configurations for different development and deployment scenarios.

## Quick Start

### Option 1: Full Development (Recommended for Frontend Development)
Backend in Docker + Frontend locally for hot reload:

```bash
# Start backend services
make dev-quick

# In another terminal, start frontend locally
cd web-ui
npm install  # first time only
npm run dev
```

Access:
- **Frontend**: http://localhost:2002 (with hot reload)
- **Backend API**: http://localhost:9080
- **Qdrant**: http://localhost:6333

### Option 2: Full Stack in Docker
Everything in Docker (slower iteration):

```bash
make full-up
```

Access:
- **WebUI**: http://localhost:2001
- **Backend API**: http://localhost:9080
- **Qdrant**: http://localhost:6333

## Available Configurations

### 1. Backend Only (`docker-compose.backend.yml`)
**Use for**: Backend development, API testing, when running frontend locally

```bash
# Start
make backend-up

# Stop
make backend-down

# Logs
make backend-logs
```

**Services**:
- Qdrant vector database
- MCP Memory Server (backend only)

**Ports**:
- `9080`: MCP API
- `9081`: Health check
- `6333`: Qdrant HTTP API
- `6334`: Qdrant gRPC API

### 2. Frontend Only (`docker-compose.frontend.yml`)
**Use for**: Frontend development when backend is running separately

```bash
# Requires backend to be running first
make backend-up
make frontend-up

# Stop
make frontend-down
```

**Services**:
- Next.js WebUI

**Ports**:
- `2001`: WebUI

### 3. Development Backend (`docker-compose.dev.yml`)
**Use for**: Frontend development with optimized backend setup

```bash
# Start
make dev-backend-up

# Stop
make dev-backend-down

# Logs
make dev-backend-logs
```

**Features**:
- Debug logging enabled
- CORS enabled for local frontend
- Separate dev volumes
- Relaxed security settings

### 4. Full Stack (`docker-compose.yml`)
**Use for**: Production-like testing, demos

```bash
# Start
make full-up

# Stop
make full-down

# Logs
make full-logs
```

**Services**:
- Everything in a single container

## Development Workflows

### Frontend Development (Fastest Iteration)
1. Start backend services: `make dev-backend-up`
2. Run frontend locally: `cd web-ui && npm run dev`
3. Develop with hot reload at http://localhost:2002

### Backend Development
1. Start only required services: `make backend-up`
2. Develop backend locally or rebuild Docker image
3. Test API at http://localhost:9080

### Full Stack Development
1. Start everything: `make full-up`
2. Access WebUI at http://localhost:2001
3. Slower iteration due to Docker rebuilds

### API Testing
1. Start backend: `make backend-up`
2. Use curl, Postman, or frontend to test API

## File Structure

```
├── docker-compose.yml           # Full stack (original)
├── docker-compose.backend.yml   # Backend only
├── docker-compose.frontend.yml  # Frontend only  
├── docker-compose.dev.yml       # Development backend
├── Dockerfile                   # Full stack image
├── Dockerfile.backend          # Backend only image
├── Dockerfile.frontend         # Frontend only image
└── Makefile                    # All commands
```

## Environment Variables

Create `.env` file in the root directory:

```bash
# Copy example
cp .env.example .env

# Edit with your settings
vim .env
```

**Key variables**:
- `OPENAI_API_KEY`: Required for embeddings
- `MCP_HOST_PORT`: Backend port (default: 9080)
- `WEBUI_PORT`: Frontend port (default: 2001)
- `QDRANT_HOST_PORT`: Qdrant port (default: 6333)

## Data Persistence

All setups use named Docker volumes for data persistence:

**Production volumes** (shared across backend/full setups):
- `mcp_memory_qdrant_vector_db_NEVER_DELETE`: Vector database
- `mcp_memory_app_data_NEVER_DELETE`: Application data
- `mcp_memory_logs_NEVER_DELETE`: Logs
- `mcp_memory_backups_NEVER_DELETE`: Backups

**Development volumes** (separate):
- `mcp_memory_qdrant_dev`: Dev vector database
- `mcp_memory_data_dev`: Dev application data

## Useful Commands

### Makefile Commands
```bash
make help                    # Show all available commands
make dev-quick              # Quick development setup
make backend-up             # Start backend only
make frontend-up            # Start frontend only
make full-up                # Start full stack
make docker-clean           # Clean up all Docker resources
make docker-rebuild         # Rebuild all images
```

### Direct Docker Compose
```bash
# Backend only
docker-compose -f docker-compose.backend.yml up -d

# Frontend only
docker-compose -f docker-compose.frontend.yml up -d

# Development backend
docker-compose -f docker-compose.dev.yml up -d

# Full stack
docker-compose up -d
```

### Health Checks
```bash
# Backend health
curl http://localhost:9081/health

# Frontend health (when running in Docker)
curl http://localhost:2001

# Qdrant health
curl http://localhost:6333/health
```

## Troubleshooting

### Frontend can't connect to backend
1. Ensure backend is running: `curl http://localhost:9080/health`
2. Check environment variables in frontend
3. Verify CORS settings in backend

### Port conflicts
1. Stop conflicting services
2. Change ports in `.env` file
3. Restart services

### Data persistence issues
1. Check Docker volumes: `docker volume ls`
2. Verify volume mounts in compose files
3. Check file permissions

### Performance issues
1. Use development setup for iteration: `make dev-quick`
2. Run frontend locally for hot reload
3. Use backend-only setup for API development

## Next Steps

- Start with `make dev-quick` for development
- Use `make backend-up` for API testing
- Use `make full-up` for demos or production-like testing
- Check logs with `make *-logs` commands
- Clean up with `make docker-clean` when needed