# Deployment Guide

This guide covers deploying the Claude Vector Memory MCP Server in various environments.

## 🚀 Deployment Options

### 1. Docker Compose (Recommended)

The easiest way to deploy with all dependencies.

#### Prerequisites
- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum
- 10GB disk space

#### Steps

1. **Clone and configure**
   ```bash
   git clone https://github.com/fredcamaral/mcp-memory.git
   cd mcp-memory
   
   # Copy and customize environment file
   cp .env.example .env
   vim .env
   ```

2. **Configure environment variables**
   ```env
   # Database
   MCP_DB_PASSWORD=your_secure_password
   MCP_REDIS_PASSWORD=your_redis_password
   
   # Security
   MCP_MASTER_KEY=your_32_byte_encryption_key
   MCP_TOKEN_SECRET=your_jwt_secret
   
   # Backup (optional)
   MCP_BACKUP_BUCKET=your-s3-bucket
   AWS_ACCESS_KEY_ID=your_access_key
   AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

3. **Deploy services**
   ```bash
   # Start all services
   docker-compose up -d
   
   # Check status
   docker-compose ps
   
   # View logs
   docker-compose logs -f mcp-memory-server
   ```

4. **Verify deployment**
   ```bash
   # Health check
   curl http://localhost:8081/health
   
   # Metrics
   curl http://localhost:8082/metrics
   
   # Grafana dashboard
   open http://localhost:3000
   # Login: admin/grafanapassword
   ```

### 2. Kubernetes Deployment

For production Kubernetes environments.

#### Prerequisites
- Kubernetes 1.20+
- kubectl configured
- Helm 3.0+ (optional)

#### Using Helm Chart (Recommended)

```bash
# Add Helm repository
helm repo add mcp-memory https://fredcamaral.github.io/mcp-memory
helm repo update

# Install with custom values
helm install mcp-memory mcp-memory/mcp-memory \
  --namespace mcp-memory \
  --create-namespace \
  -f values-production.yaml
```

#### Manual Kubernetes Deployment

```bash
# Create namespace
kubectl create namespace mcp-memory

# Create secrets
kubectl create secret generic mcp-memory-secrets \
  --from-literal=db-password='your_password' \
  --from-literal=master-key='your_encryption_key' \
  --namespace mcp-memory

# Apply manifests
kubectl apply -f k8s/ -n mcp-memory

# Check deployment
kubectl get pods -n mcp-memory
kubectl logs -f deployment/mcp-memory-server -n mcp-memory
```

### 3. Standalone Docker

For simple single-container deployments.

```bash
# Create data directory
mkdir -p ./data

# Run container
docker run -d \
  --name mcp-memory-server \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 8082:8082 \
  -v $(pwd)/data:/app/data \
  -e MCP_MEMORY_LOG_LEVEL=info \
  mcp-memory-server:latest

# Check logs
docker logs -f mcp-memory-server
```

### 4. Binary Installation

For direct host installation.

```bash
# Download binary
wget https://github.com/fredcamaral/mcp-memory/releases/latest/download/mcp-memory-server-linux-amd64.tar.gz
tar -xzf mcp-memory-server-linux-amd64.tar.gz

# Install binary
sudo mv mcp-memory-server /usr/local/bin/
sudo chmod +x /usr/local/bin/mcp-memory-server

# Create user and directories
sudo useradd --system --shell /bin/false mcp-memory
sudo mkdir -p /var/lib/mcp-memory /etc/mcp-memory /var/log/mcp-memory
sudo chown mcp-memory:mcp-memory /var/lib/mcp-memory /var/log/mcp-memory

# Copy configuration
sudo cp configs/production/config.yaml /etc/mcp-memory/

# Create systemd service
sudo tee /etc/systemd/system/mcp-memory.service > /dev/null <<EOF
[Unit]
Description=MCP Memory Server
After=network.target

[Service]
Type=simple
User=mcp-memory
Group=mcp-memory
ExecStart=/usr/local/bin/mcp-memory-server --config /etc/mcp-memory/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable mcp-memory
sudo systemctl start mcp-memory
sudo systemctl status mcp-memory
```

## 🔧 Environment Configuration

### Development Environment

```yaml
# configs/dev/config.yaml
server:
  host: "localhost"
  port: 8080

logging:
  level: "debug"
  format: "text"

storage:
  type: "sqlite"
  sqlite:
    path: "./data/memory_dev.db"

security:
  encryption:
    enabled: false
  access_control:
    enabled: false
```

### Staging Environment

```yaml
# configs/staging/config.yaml
server:
  host: "0.0.0.0"
  port: 8080

logging:
  level: "info"
  format: "json"

storage:
  type: "postgres"
  postgres:
    host: "postgres-staging"
    database: "mcp_memory_staging"
    username: "mcpuser"
    password: "${MCP_DB_PASSWORD}"

security:
  encryption:
    enabled: true
  access_control:
    enabled: true
```

### Production Environment

```yaml
# configs/production/config.yaml
server:
  host: "0.0.0.0"
  port: 8080

logging:
  level: "warn"
  format: "json"

storage:
  type: "postgres"
  postgres:
    host: "${MCP_DB_HOST}"
    database: "${MCP_DB_NAME}"
    username: "${MCP_DB_USER}"
    password: "${MCP_DB_PASSWORD}"
    ssl_mode: "require"
    max_connections: 50

vector:
  cache_size: 10000
  persist_path: "/app/data/vectors"

security:
  encryption:
    enabled: true
    master_key_env: "MCP_MASTER_KEY"
  access_control:
    enabled: true
    token_secret: "${MCP_TOKEN_SECRET}"
  rate_limiting:
    enabled: true

backup:
  enabled: true
  interval: 24h
  s3:
    enabled: true
    bucket: "${MCP_BACKUP_BUCKET}"
```

## 🗄️ Database Setup

### PostgreSQL (Recommended for Production)

```bash
# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE mcp_memory;
CREATE USER mcpuser WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE mcp_memory TO mcpuser;
ALTER USER mcpuser CREATEDB;
EOF

# Configure PostgreSQL
echo "host mcp_memory mcpuser 0.0.0.0/0 md5" >> /etc/postgresql/*/main/pg_hba.conf
echo "listen_addresses = '*'" >> /etc/postgresql/*/main/postgresql.conf

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### SQLite (Development/Small Deployments)

```bash
# Create data directory
mkdir -p /var/lib/mcp-memory

# Ensure permissions
chown mcp-memory:mcp-memory /var/lib/mcp-memory
chmod 755 /var/lib/mcp-memory
```

## 🔐 Security Configuration

### SSL/TLS Setup

#### Using Let's Encrypt with Traefik

```yaml
# docker-compose.yml
traefik:
  command:
    - --certificatesResolvers.letsencrypt.acme.email=admin@yourdomain.com
    - --certificatesResolvers.letsencrypt.acme.storage=/etc/traefik/certs/acme.json
    - --certificatesResolvers.letsencrypt.acme.httpChallenge.entryPoint=web

mcp-memory-server:
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.mcp-memory.rule=Host(`mcp.yourdomain.com`)"
    - "traefik.http.routers.mcp-memory.tls=true"
    - "traefik.http.routers.mcp-memory.tls.certresolver=letsencrypt"
```

#### Manual SSL Certificate

```bash
# Generate self-signed certificate (development only)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Use existing certificate
docker run -d \
  -v /path/to/cert.pem:/app/cert.pem \
  -v /path/to/key.pem:/app/key.pem \
  -e MCP_MEMORY_SSL_ENABLED=true \
  -e MCP_MEMORY_SSL_CERT=/app/cert.pem \
  -e MCP_MEMORY_SSL_KEY=/app/key.pem \
  mcp-memory-server:latest
```

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 8080/tcp  # MCP Memory API (optional, if direct access needed)
sudo ufw enable

# iptables
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

## 📊 Monitoring Setup

### Prometheus Configuration

```yaml
# configs/prometheus/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'mcp-memory-server'
    static_configs:
      - targets: ['mcp-memory-server:8082']
    scrape_interval: 15s
```

### Grafana Dashboard Import

1. **Access Grafana**: http://localhost:3000
2. **Login**: admin/grafanapassword
3. **Import Dashboard**: Use the provided dashboard JSON files in `configs/grafana/dashboards/`
4. **Configure Alerts**: Set up alert rules for critical metrics

### Alert Manager

```yaml
# configs/alertmanager/alertmanager.yml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@yourdomain.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
  - name: 'web.hook'
    email_configs:
      - to: 'admin@yourdomain.com'
        subject: 'MCP Memory Alert: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
```

## 🔄 Backup and Restore

### Automated Backups

```bash
# Configure backup in production config
backup:
  enabled: true
  interval: 24h
  retention_days: 30
  path: "/app/backups"
  s3:
    enabled: true
    bucket: "mcp-memory-backups"
    region: "us-west-2"
```

### Manual Backup

```bash
# Create backup
curl -X POST http://localhost:8080/api/backup

# Download backup
curl -O http://localhost:8080/api/backups/backup-20241201-120000.tar.gz

# Restore from backup
curl -X POST http://localhost:8080/api/restore \
  -H "Content-Type: application/json" \
  -d '{"backup_id": "backup-20241201-120000"}'
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Container Won't Start

```bash
# Check logs
docker logs mcp-memory-server

# Common causes:
# - Missing environment variables
# - Database connection issues
# - Port conflicts
# - Insufficient permissions
```

#### 2. Database Connection Failed

```bash
# Test database connectivity
docker exec -it mcp-postgres psql -U mcpuser -d mcp_memory -c "SELECT 1;"

# Check environment variables
docker exec mcp-memory-server env | grep MCP_
```

#### 3. High Memory Usage

```bash
# Check memory metrics
curl http://localhost:8082/metrics | grep memory

# Adjust cache sizes in config
caching:
  memory:
    size: 1000  # Reduce from default
  vector:
    size: 100   # Reduce from default
```

#### 4. Vector Search Performance

```bash
# Check vector cache hit rate
curl http://localhost:8082/metrics | grep cache_hit

# Optimize vector configuration
vector:
  cache_size: 10000     # Increase cache
  nlist: 1000          # Increase for better accuracy
  nprobe: 50           # Increase for better recall
```

### Health Checks

```bash
# Application health
curl http://localhost:8081/health

# Component health
curl http://localhost:8081/health/database
curl http://localhost:8081/health/vector
curl http://localhost:8081/health/memory

# Metrics endpoint
curl http://localhost:8082/metrics
```

### Log Analysis

```bash
# View application logs
docker logs -f mcp-memory-server

# Filter by log level
docker logs mcp-memory-server 2>&1 | grep "ERROR"

# Follow specific operations
docker logs mcp-memory-server 2>&1 | grep "vector_search"
```

## 📈 Performance Tuning

### Resource Requirements

| Environment | CPU | Memory | Storage |
|-------------|-----|--------|---------|
| Development | 1 core | 512MB | 5GB |
| Staging | 2 cores | 2GB | 20GB |
| Production | 4+ cores | 4GB+ | 100GB+ |

### Optimization Tips

1. **Database Connection Pooling**
   ```yaml
   storage:
     postgres:
       max_connections: 50
       max_idle_connections: 25
   ```

2. **Vector Cache Tuning**
   ```yaml
   vector:
     cache_size: 10000  # Adjust based on memory
   ```

3. **Memory Management**
   ```yaml
   memory:
     cleanup_interval: 30m  # More frequent cleanup
     max_memory_entries: 50000  # Limit entries
   ```

4. **Caching Strategy**
   ```yaml
   caching:
     memory:
       type: "lru"
       size: 10000
     query:
       type: "lfu"
       size: 5000
   ```

## 🔄 Updates and Maintenance

### Rolling Updates

```bash
# Docker Compose
docker-compose pull
docker-compose up -d

# Kubernetes
kubectl rollout restart deployment/mcp-memory-server
kubectl rollout status deployment/mcp-memory-server
```

### Database Migrations

```bash
# Check migration status
docker exec mcp-memory-server /app/mcp-memory-server migrate status

# Run migrations
docker exec mcp-memory-server /app/mcp-memory-server migrate up
```

### Maintenance Mode

```bash
# Enable maintenance mode
curl -X POST http://localhost:8080/api/maintenance/enable

# Disable maintenance mode
curl -X POST http://localhost:8080/api/maintenance/disable
```

---

This deployment guide should get you up and running in any environment. For additional help, check the [troubleshooting section](TROUBLESHOOTING.md) or [open an issue](https://github.com/fredcamaral/mcp-memory/issues).