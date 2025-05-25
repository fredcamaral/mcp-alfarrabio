# Optional Monitoring and Infrastructure Services

This directory contains configuration files for optional monitoring and infrastructure services that can enhance your MCP Memory deployment.

## Available Services

### 1. Prometheus (Metrics Collection)
- **Directory**: `prometheus/`
- **Purpose**: Collects metrics from the MCP Memory server
- **Files**: 
  - `prometheus.yml` - Prometheus server configuration
  - `rules/alerts.yml` - Alerting rules

### 2. Grafana (Metrics Visualization)
- **Directory**: `grafana/`
- **Purpose**: Visualizes metrics collected by Prometheus
- **Files**:
  - `dashboards/mcp-memory-overview.json` - Pre-built dashboard
  - `provisioning/` - Auto-provisioning configuration

### 3. Traefik (Reverse Proxy)
- **Directory**: `traefik/`
- **Purpose**: Load balancing and SSL termination
- **Files**:
  - `traefik.yml` - Main configuration
  - `dynamic/` - Dynamic configuration

## Enabling Optional Services

To enable these services, create a `docker-compose.override.yml` file with the desired services:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: mcp-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./configs/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    networks:
      - mcp_network

  grafana:
    image: grafana/grafana:latest
    container_name: mcp-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD:-grafanapassword}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./configs/grafana/provisioning:/etc/grafana/provisioning:ro
      - ./configs/grafana/dashboards:/var/lib/grafana/dashboards:ro
    networks:
      - mcp_network
    depends_on:
      - prometheus

  traefik:
    image: traefik:v3.0
    container_name: mcp-traefik
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./configs/traefik/traefik.yml:/etc/traefik/traefik.yml:ro
    networks:
      - mcp_network

volumes:
  prometheus_data:
    driver: local
  grafana_data:
    driver: local
  traefik_certs:
    driver: local
```

## Note on Config Files

The MCP Memory server itself doesn't use YAML configuration files. It relies entirely on environment variables for configuration. The YAML files in `dev/`, `docker/`, `production/`, and `staging/` directories are kept as documentation templates showing recommended environment variable settings for different deployment scenarios.