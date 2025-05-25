# MCP Memory Server Deployment Guide

This comprehensive guide covers deploying the MCP Memory Server in production environments using Docker, Kubernetes, and cloud platforms.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Deployments](#cloud-deployments)
- [Configuration Management](#configuration-management)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)
- [Backup and Recovery](#backup-and-recovery)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Docker 20.10+ or Podman 3.0+
- Kubernetes 1.24+ (for K8s deployment)
- Helm 3.0+ (for K8s deployment)
- Prometheus/Grafana (for monitoring)
- PostgreSQL 14+ or ChromaDB
- OpenAI API key or compatible embedding service

## Docker Deployment

### Building the Docker Image

```dockerfile
# Multi-stage build for optimal size
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN make build

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/bin/server /usr/local/bin/mcp-memory

# Copy default config
COPY --from=builder /app/configs/production/config.yaml /etc/mcp-memory/config.yaml

# Set ownership
RUN chown -R appuser:appuser /etc/mcp-memory

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/usr/local/bin/mcp-memory", "health"]

# Run the server
ENTRYPOINT ["/usr/local/bin/mcp-memory"]
CMD ["--config", "/etc/mcp-memory/config.yaml"]
```

### Docker Compose Deployment

```yaml
version: '3.8'

services:
  mcp-memory:
    image: your-registry/mcp-memory:latest
    container_name: mcp-memory
    restart: unless-stopped
    ports:
      - "8080:8080"  # HTTP API
      - "9090:9090"  # Metrics
    environment:
      - MCP_MEMORY_ENV=production
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - LOG_LEVEL=info
    volumes:
      - ./config:/etc/mcp-memory:ro
      - mcp-data:/var/lib/mcp-memory
    networks:
      - mcp-network
    depends_on:
      - chroma
      - postgres
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G

  chroma:
    image: chromadb/chroma:latest
    container_name: chroma
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - chroma-data:/chroma/chroma
    environment:
      - IS_PERSISTENT=TRUE
      - ANONYMIZED_TELEMETRY=FALSE
    networks:
      - mcp-network

  postgres:
    image: postgres:15-alpine
    container_name: postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=mcp_memory
      - POSTGRES_USER=mcp
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./scripts/init-postgres.sql:/docker-entrypoint-initdb.d/init.sql:ro
    networks:
      - mcp-network

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: unless-stopped
    ports:
      - "9091:9090"
    volumes:
      - ./configs/prometheus:/etc/prometheus:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    networks:
      - mcp-network

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - ./configs/grafana:/etc/grafana/provisioning:ro
      - grafana-data:/var/lib/grafana
    networks:
      - mcp-network

volumes:
  mcp-data:
  chroma-data:
  postgres-data:
  prometheus-data:
  grafana-data:

networks:
  mcp-network:
    driver: bridge
```

### Production Docker Best Practices

```bash
#!/bin/bash
# deploy-docker.sh

# Build with buildkit for better caching
export DOCKER_BUILDKIT=1

# Build image with version tag
VERSION=$(git describe --tags --always)
docker build -t mcp-memory:${VERSION} -t mcp-memory:latest .

# Security scan
docker scan mcp-memory:${VERSION}

# Push to registry
docker push your-registry/mcp-memory:${VERSION}
docker push your-registry/mcp-memory:latest

# Deploy with rolling update
docker-compose up -d --no-deps --scale mcp-memory=2 mcp-memory

# Wait for health check
./scripts/wait-for-healthy.sh mcp-memory

# Scale down old containers
docker-compose up -d --no-deps --scale mcp-memory=1 mcp-memory
```

## Kubernetes Deployment

### Kubernetes Manifests

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: mcp-memory
---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-memory-config
  namespace: mcp-memory
data:
  config.yaml: |
    environment: production
    service:
      name: mcp-memory
      version: 1.0.0
    
    server:
      mode: http
      port: 8080
      timeout: 30s
      cors:
        enabled: true
        origins: ["*"]
    
    chroma:
      url: "http://chroma:8000"
      collection: "memory_production"
    
    openai:
      model: "text-embedding-3-small"
      max_retries: 3
      timeout: 30s
    
    features:
      auto_analysis: true
      pattern_detection: true
      context_suggestions: true
      multi_repository: true
    
    monitoring:
      enabled: true
      port: 9090
      path: /metrics
---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: mcp-memory-secrets
  namespace: mcp-memory
type: Opaque
stringData:
  openai-api-key: "your-openai-api-key"
  postgres-password: "your-postgres-password"
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-memory
  namespace: mcp-memory
  labels:
    app: mcp-memory
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mcp-memory
  template:
    metadata:
      labels:
        app: mcp-memory
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: mcp-memory
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: mcp-memory
        image: your-registry/mcp-memory:latest
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        - name: metrics
          containerPort: 9090
          protocol: TCP
        env:
        - name: MCP_MEMORY_ENV
          value: "production"
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: mcp-memory-secrets
              key: openai-api-key
        - name: LOG_LEVEL
          value: "info"
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: config
          mountPath: /etc/mcp-memory
          readOnly: true
        - name: data
          mountPath: /var/lib/mcp-memory
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 2Gi
      volumes:
      - name: config
        configMap:
          name: mcp-memory-config
      - name: data
        persistentVolumeClaim:
          claimName: mcp-memory-data
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: mcp-memory
  namespace: mcp-memory
  labels:
    app: mcp-memory
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
  - name: metrics
    port: 9090
    targetPort: metrics
    protocol: TCP
  selector:
    app: mcp-memory
---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mcp-memory
  namespace: mcp-memory
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  tls:
  - hosts:
    - mcp-memory.example.com
    secretName: mcp-memory-tls
  rules:
  - host: mcp-memory.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: mcp-memory
            port:
              number: 8080
---
# pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mcp-memory-data
  namespace: mcp-memory
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
---
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: mcp-memory
  namespace: mcp-memory
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mcp-memory
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: mcp_memory_active_connections
      target:
        type: AverageValue
        averageValue: "100"
---
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mcp-memory
  namespace: mcp-memory
---
# role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: mcp-memory
  namespace: mcp-memory
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
---
# rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mcp-memory
  namespace: mcp-memory
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: mcp-memory
subjects:
- kind: ServiceAccount
  name: mcp-memory
  namespace: mcp-memory
```

### Helm Chart

```yaml
# Chart.yaml
apiVersion: v2
name: mcp-memory
description: MCP Memory Server Helm chart
type: application
version: 1.0.0
appVersion: "1.0.0"

# values.yaml
replicaCount: 3

image:
  repository: your-registry/mcp-memory
  pullPolicy: IfNotPresent
  tag: "latest"

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"

podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

securityContext:
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false

service:
  type: ClusterIP
  port: 8080
  metricsPort: 9090

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
  hosts:
    - host: mcp-memory.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mcp-memory-tls
      hosts:
        - mcp-memory.example.com

resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 1Gi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

persistence:
  enabled: true
  storageClass: "fast-ssd"
  accessMode: ReadWriteOnce
  size: 10Gi

config:
  environment: production
  server:
    mode: http
    port: 8080
    timeout: 30s
  chroma:
    url: "http://chroma:8000"
    collection: "memory_production"
  openai:
    model: "text-embedding-3-small"
  features:
    autoAnalysis: true
    patternDetection: true
    contextSuggestions: true
    multiRepository: true

secrets:
  openaiApiKey: ""
  postgresPassword: ""

chroma:
  enabled: true
  replicaCount: 1
  persistence:
    enabled: true
    size: 20Gi

postgresql:
  enabled: true
  auth:
    database: mcp_memory
    username: mcp
  primary:
    persistence:
      enabled: true
      size: 20Gi

prometheus:
  enabled: true

grafana:
  enabled: true
  adminPassword: "changeme"
```

### Helm Deployment

```bash
#!/bin/bash
# deploy-helm.sh

# Add helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Create namespace
kubectl create namespace mcp-memory

# Install dependencies
helm install chroma ./charts/chroma -n mcp-memory
helm install postgresql bitnami/postgresql -n mcp-memory \
  --set auth.database=mcp_memory \
  --set auth.username=mcp

# Install MCP Memory
helm install mcp-memory ./charts/mcp-memory -n mcp-memory \
  --set secrets.openaiApiKey=$OPENAI_API_KEY \
  --set secrets.postgresPassword=$POSTGRES_PASSWORD \
  --set image.tag=$(git describe --tags --always)

# Wait for deployment
kubectl rollout status deployment/mcp-memory -n mcp-memory

# Install monitoring stack
helm install prometheus prometheus-community/kube-prometheus-stack -n mcp-memory
helm install grafana grafana/grafana -n mcp-memory \
  --set adminPassword=$GRAFANA_PASSWORD
```

## Cloud Deployments

### AWS EKS Deployment

```bash
#!/bin/bash
# deploy-eks.sh

# Create EKS cluster
eksctl create cluster \
  --name mcp-memory-cluster \
  --region us-west-2 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 2 \
  --nodes-max 10 \
  --managed

# Install AWS Load Balancer Controller
helm install aws-load-balancer-controller \
  eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=mcp-memory-cluster

# Create storage class for EBS
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
volumeBindingMode: WaitForFirstConsumer
EOF

# Deploy application
helm install mcp-memory ./charts/mcp-memory \
  --set service.type=LoadBalancer \
  --set persistence.storageClass=fast-ssd
```

### Google GKE Deployment

```bash
#!/bin/bash
# deploy-gke.sh

# Create GKE cluster
gcloud container clusters create mcp-memory-cluster \
  --zone us-central1-a \
  --num-nodes 3 \
  --enable-autoscaling \
  --min-nodes 2 \
  --max-nodes 10 \
  --machine-type n2-standard-2 \
  --enable-stackdriver-kubernetes

# Get credentials
gcloud container clusters get-credentials mcp-memory-cluster \
  --zone us-central1-a

# Create storage class for SSD
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  replication-type: regional-pd
volumeBindingMode: WaitForFirstConsumer
EOF

# Deploy with Google Cloud Load Balancer
helm install mcp-memory ./charts/mcp-memory \
  --set ingress.annotations."kubernetes\.io/ingress\.class"=gce \
  --set persistence.storageClass=fast-ssd
```

### Azure AKS Deployment

```bash
#!/bin/bash
# deploy-aks.sh

# Create resource group
az group create --name mcp-memory-rg --location eastus

# Create AKS cluster
az aks create \
  --resource-group mcp-memory-rg \
  --name mcp-memory-cluster \
  --node-count 3 \
  --enable-cluster-autoscaler \
  --min-count 2 \
  --max-count 10 \
  --node-vm-size Standard_DS2_v2 \
  --enable-managed-identity

# Get credentials
az aks get-credentials \
  --resource-group mcp-memory-rg \
  --name mcp-memory-cluster

# Create storage class for Azure Disk
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/azure-disk
parameters:
  storageaccounttype: Premium_LRS
  kind: Managed
volumeBindingMode: WaitForFirstConsumer
EOF

# Deploy application
helm install mcp-memory ./charts/mcp-memory \
  --set persistence.storageClass=fast-ssd
```

## Configuration Management

### Environment-Specific Configs

```yaml
# base/config.yaml
service:
  name: mcp-memory
  version: ${VERSION}

server:
  port: ${PORT:8080}
  timeout: ${TIMEOUT:30s}

logging:
  level: ${LOG_LEVEL:info}
  format: ${LOG_FORMAT:json}

# production/config.yaml
include:
  - base/config.yaml

environment: production

server:
  mode: http
  cors:
    enabled: true
    origins: ${CORS_ORIGINS:["https://app.example.com"]}

features:
  rate_limiting:
    enabled: true
    requests_per_minute: 100
  
  caching:
    enabled: true
    ttl: 300s

monitoring:
  enabled: true
  detailed_metrics: true
```

### Secret Management

```bash
#!/bin/bash
# setup-secrets.sh

# Using Kubernetes Secrets
kubectl create secret generic mcp-memory-secrets \
  --from-literal=openai-api-key=$OPENAI_API_KEY \
  --from-literal=postgres-password=$POSTGRES_PASSWORD \
  -n mcp-memory

# Using Sealed Secrets
echo -n $OPENAI_API_KEY | kubectl create secret generic mcp-memory-secrets \
  --dry-run=client \
  --from-literal=openai-api-key=/dev/stdin \
  -o yaml | kubeseal -o yaml > sealed-secrets.yaml

# Using AWS Secrets Manager
aws secretsmanager create-secret \
  --name mcp-memory/openai-api-key \
  --secret-string $OPENAI_API_KEY

# Using HashiCorp Vault
vault kv put secret/mcp-memory/openai-api-key value=$OPENAI_API_KEY
```

## Monitoring and Observability

### Prometheus Configuration

```yaml
# prometheus-config.yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'mcp-memory'
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - mcp-memory
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
    - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
      action: replace
      regex: ([^:]+)(?::\d+)?;(\d+)
      replacement: $1:$2
      target_label: __address__

rule_files:
  - 'alerts.yml'

alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alertmanager:9093
```

### Grafana Dashboards

```json
{
  "dashboard": {
    "title": "MCP Memory Server",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(mcp_memory_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(mcp_memory_errors_total[5m])"
          }
        ]
      },
      {
        "title": "Response Time",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(mcp_memory_request_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Active Sessions",
        "targets": [
          {
            "expr": "mcp_memory_active_sessions"
          }
        ]
      }
    ]
  }
}
```

### Logging Configuration

```yaml
# fluent-bit-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: mcp-memory
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         1
        Log_Level     info
        Daemon        off

    [INPUT]
        Name              tail
        Path              /var/log/containers/*mcp-memory*.log
        Parser            docker
        Tag               mcp.memory.*
        Refresh_Interval  5

    [FILTER]
        Name         parser
        Match        mcp.memory.*
        Key_Name     log
        Parser       json

    [OUTPUT]
        Name         es
        Match        mcp.memory.*
        Host         elasticsearch
        Port         9200
        Index        mcp-memory
        Type         _doc
```

## Security Hardening

### Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mcp-memory-network-policy
  namespace: mcp-memory
spec:
  podSelector:
    matchLabels:
      app: mcp-memory
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    - podSelector:
        matchLabels:
          app: prometheus
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 9090
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: chroma
    ports:
    - protocol: TCP
      port: 8000
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

### Pod Security Policy

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: mcp-memory-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

## Performance Tuning

### Resource Optimization

```yaml
# resource-tuning.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-memory-tuning
  namespace: mcp-memory
data:
  GOGC: "100"
  GOMEMLIMIT: "1900MiB"
  GOMAXPROCS: "2"
  
  # Connection pooling
  MAX_CONNECTIONS: "100"
  IDLE_CONNECTIONS: "10"
  CONNECTION_TIMEOUT: "30s"
  
  # Caching
  CACHE_SIZE: "1000"
  CACHE_TTL: "300s"
  
  # Batch processing
  BATCH_SIZE: "100"
  BATCH_TIMEOUT: "5s"
```

### Database Optimization

```sql
-- PostgreSQL performance tuning
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;

-- Create indexes
CREATE INDEX idx_chunks_session_id ON memory_chunks(session_id);
CREATE INDEX idx_chunks_repository ON memory_chunks(repository);
CREATE INDEX idx_chunks_timestamp ON memory_chunks(timestamp);
CREATE INDEX idx_chunks_embedding ON memory_chunks USING ivfflat (embedding vector_cosine_ops);

-- Partitioning for large datasets
CREATE TABLE memory_chunks_2024_01 PARTITION OF memory_chunks
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## Backup and Recovery

### Automated Backups

```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mcp-memory-backup
  namespace: mcp-memory
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: your-registry/mcp-backup:latest
            env:
            - name: POSTGRES_HOST
              value: postgresql
            - name: POSTGRES_DB
              value: mcp_memory
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: postgresql
                  key: username
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgresql
                  key: password
            - name: S3_BUCKET
              value: mcp-memory-backups
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: access-key-id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: secret-access-key
            command:
            - /bin/bash
            - -c
            - |
              # Backup PostgreSQL
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)
              pg_dump -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB | \
                gzip > /tmp/postgres_backup_$TIMESTAMP.sql.gz
              
              # Backup Chroma
              curl -X POST http://chroma:8000/api/v1/collections/memory_production/export \
                -o /tmp/chroma_backup_$TIMESTAMP.json
              
              # Upload to S3
              aws s3 cp /tmp/postgres_backup_$TIMESTAMP.sql.gz \
                s3://$S3_BUCKET/postgres/
              aws s3 cp /tmp/chroma_backup_$TIMESTAMP.json \
                s3://$S3_BUCKET/chroma/
              
              # Clean up old backups (keep last 30 days)
              aws s3 ls s3://$S3_BUCKET/postgres/ | \
                awk '{print $4}' | \
                sort -r | \
                tail -n +31 | \
                xargs -I {} aws s3 rm s3://$S3_BUCKET/postgres/{}
          restartPolicy: OnFailure
```

### Disaster Recovery

```bash
#!/bin/bash
# disaster-recovery.sh

# Restore PostgreSQL
LATEST_POSTGRES_BACKUP=$(aws s3 ls s3://mcp-memory-backups/postgres/ | \
  sort | tail -n 1 | awk '{print $4}')

aws s3 cp s3://mcp-memory-backups/postgres/$LATEST_POSTGRES_BACKUP - | \
  gunzip | \
  kubectl exec -i postgresql-0 -n mcp-memory -- \
    psql -U mcp -d mcp_memory

# Restore Chroma
LATEST_CHROMA_BACKUP=$(aws s3 ls s3://mcp-memory-backups/chroma/ | \
  sort | tail -n 1 | awk '{print $4}')

aws s3 cp s3://mcp-memory-backups/chroma/$LATEST_CHROMA_BACKUP /tmp/
curl -X POST http://chroma:8000/api/v1/collections/memory_production/import \
  -H "Content-Type: application/json" \
  -d @/tmp/$LATEST_CHROMA_BACKUP

# Verify restoration
kubectl exec -it deployment/mcp-memory -n mcp-memory -- \
  /usr/local/bin/mcp-memory verify --config /etc/mcp-memory/config.yaml
```

## Troubleshooting

### Common Issues and Solutions

#### 1. High Memory Usage

```bash
# Check memory usage
kubectl top pods -n mcp-memory

# Get heap profile
kubectl exec -it deployment/mcp-memory -n mcp-memory -- \
  curl -s http://localhost:6060/debug/pprof/heap > heap.prof

# Analyze with pprof
go tool pprof heap.prof
```

#### 2. Slow Queries

```bash
# Enable query logging
kubectl exec -it postgresql-0 -n mcp-memory -- \
  psql -U mcp -d mcp_memory -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"

# Check slow queries
kubectl logs postgresql-0 -n mcp-memory | grep "duration:"

# Analyze query plan
kubectl exec -it postgresql-0 -n mcp-memory -- \
  psql -U mcp -d mcp_memory -c "EXPLAIN ANALYZE SELECT ..."
```

#### 3. Connection Issues

```bash
# Test connectivity
kubectl run debug --image=busybox -it --rm --restart=Never -- \
  /bin/sh -c "wget -O- http://mcp-memory:8080/health"

# Check DNS resolution
kubectl run debug --image=busybox -it --rm --restart=Never -- \
  nslookup mcp-memory.mcp-memory.svc.cluster.local

# Verify network policies
kubectl describe networkpolicy -n mcp-memory
```

#### 4. Deployment Failures

```bash
# Check deployment status
kubectl describe deployment mcp-memory -n mcp-memory

# View pod events
kubectl describe pod -l app=mcp-memory -n mcp-memory

# Check logs
kubectl logs -l app=mcp-memory -n mcp-memory --tail=100

# Debug with ephemeral container
kubectl debug -it deployment/mcp-memory -n mcp-memory \
  --image=busybox --target=mcp-memory
```

### Health Check Endpoints

```bash
# Liveness check
curl http://mcp-memory:8080/health

# Readiness check
curl http://mcp-memory:8080/ready

# Detailed health status
curl http://mcp-memory:8080/health/detailed
```

## Best Practices Summary

1. **Use GitOps**: Store all configurations in Git and use tools like ArgoCD or Flux
2. **Enable Monitoring**: Always deploy with Prometheus and Grafana
3. **Implement Proper Logging**: Use structured logging and centralized log aggregation
4. **Security First**: Enable network policies, use least privilege, scan images
5. **Automate Everything**: Use CI/CD pipelines for deployments
6. **Plan for Failure**: Implement proper backup and disaster recovery
7. **Performance Testing**: Load test before production deployment
8. **Documentation**: Keep runbooks and operational procedures updated

## Conclusion

This deployment guide provides comprehensive instructions for deploying MCP Memory Server in production environments. Follow the best practices and security guidelines to ensure a robust, scalable, and secure deployment.