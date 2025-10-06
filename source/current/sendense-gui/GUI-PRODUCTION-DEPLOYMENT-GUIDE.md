# Sendense GUI Production Deployment Guide

## 🚀 Production Deployment Overview

This guide covers deploying the Sendense GUI to production servers after successful Phase 8 completion.

## 📋 Prerequisites

### **System Requirements**
- Node.js 18+ (tested with Node.js 19.1.0)
- npm or yarn package manager
- Linux server (Ubuntu 20.04+ recommended)
- Minimum 2GB RAM, 10GB disk space

### **Build Verification**
Before deployment, ensure:
```bash
cd /home/oma_admin/sendense/source/current/sendense-gui

# Verify development mode works
npm run dev  # Should start on http://localhost:3000

# Verify production build succeeds
npm run build  # Should complete without errors

# Verify production server works
npm run start  # Should serve on http://localhost:3000
```

## 🏗️ Build Process

### **1. Clean Build**
```bash
cd /home/oma_admin/sendense/source/current/sendense-gui

# Clean previous builds
rm -rf .next

# Install dependencies
npm ci --production=false

# Build for production
npm run build
```

### **2. Build Verification**
```bash
# Check build output
ls -la .next/

# Verify static generation
ls -la .next/static/

# Check build size
du -sh .next/
```

Expected output:
```
.next/
├── static/
│   ├── chunks/
│   └── _next/static/
└── server/
    └── app/
        ├── dashboard/
        ├── protection-flows/
        ├── report-center/
        └── ...
```

## 🚀 Deployment Methods

### **Method 1: Node.js Production Server (Recommended)**

#### **1. Create Production Directory**
```bash
# Create production directory
sudo mkdir -p /opt/sendense-gui
sudo chown oma_admin:oma_admin /opt/sendense-gui

# Copy built application
cd /home/oma_admin/sendense/source/current/sendense-gui
cp -r .next /opt/sendense-gui/
cp -r public /opt/sendense-gui/
cp package.json package-lock.json next.config.ts /opt/sendense-gui/
```

#### **2. Install Production Dependencies**
```bash
cd /opt/sendense-gui

# Install only production dependencies
npm ci --production=true
```

#### **3. Create systemd Service**
```bash
sudo tee /etc/systemd/system/sendense-gui.service > /dev/null <<EOF
[Unit]
Description=Sendense Professional GUI
After=network.target

[Service]
Type=simple
User=oma_admin
WorkingDirectory=/opt/sendense-gui
ExecStart=/usr/bin/npm run start
Restart=always
RestartSec=10
Environment=NODE_ENV=production
Environment=PORT=3001

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectHome=yes
ProtectSystem=strict
ReadWritePaths=/opt/sendense-gui

[Install]
WantedBy=multi-user.target
EOF
```

#### **4. Configure Environment Variables**
```bash
# Create environment file
sudo tee /opt/sendense-gui/.env.production > /dev/null <<EOF
NODE_ENV=production
PORT=3001

# Add your production environment variables here
# NEXT_PUBLIC_API_URL=https://api.sendense.com
# DATABASE_URL=postgresql://...
EOF

# Update systemd service to use env file
sudo sed -i '/Environment=NODE_ENV=production/a EnvironmentFile=/opt/sendense-gui/.env.production' /etc/systemd/system/sendense-gui.service
```

#### **5. Start Service**
```bash
# Reload systemd and start service
sudo systemctl daemon-reload
sudo systemctl enable sendense-gui
sudo systemctl start sendense-gui

# Check status
sudo systemctl status sendense-gui

# Check logs
sudo journalctl -u sendense-gui -f
```

### **Method 2: Docker Deployment**

#### **1. Create Dockerfile**
```dockerfile
FROM node:18-alpine AS base

# Install dependencies only when needed
FROM base AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./
RUN npm ci --only=production

# Build the application
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build the application
RUN npm run build

# Production image
FROM base AS runner
WORKDIR /app

ENV NODE_ENV production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT 3000

CMD ["node", "server.js"]
```

#### **2. Build and Run**
```bash
# Build Docker image
docker build -t sendense-gui .

# Run container
docker run -d \
  --name sendense-gui \
  -p 3001:3000 \
  --restart unless-stopped \
  sendense-gui
```

## 🔧 Configuration

### **Environment Variables**
```bash
# Production environment file (.env.production)
NODE_ENV=production
PORT=3001

# API Configuration
NEXT_PUBLIC_API_URL=https://api.sendense.com
NEXT_PUBLIC_WS_URL=wss://api.sendense.com

# Database (if needed for static generation)
DATABASE_URL=postgresql://user:pass@host:5432/db

# Security
NEXTAUTH_SECRET=your-secret-key
NEXTAUTH_URL=https://gui.sendense.com
```

### **Reverse Proxy (nginx)**
```nginx
# /etc/nginx/sites-available/sendense-gui
server {
    listen 80;
    server_name gui.sendense.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name gui.sendense.com;

    # SSL configuration
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header Referrer-Policy strict-origin-when-cross-origin;

    # Gzip compression
    gzip on;
    gzip_types text/css application/javascript application/json;

    location / {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Timeout settings
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Static file caching
    location /_next/static {
        proxy_pass http://localhost:3001;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

## 📊 Monitoring and Maintenance

### **Health Checks**
```bash
# Service health
sudo systemctl status sendense-gui

# Application health
curl -f http://localhost:3001/api/health

# Logs
sudo journalctl -u sendense-gui --since "1 hour ago"
```

### **Log Rotation**
```bash
# Configure log rotation
sudo tee /etc/logrotate.d/sendense-gui > /dev/null <<EOF
/opt/sendense-gui/logs/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 oma_admin oma_admin
    postrotate
        systemctl reload sendense-gui
    endscript
}
EOF
```

### **Backup Strategy**
```bash
# Backup script
#!/bin/bash
BACKUP_DIR="/opt/backups/sendense-gui"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup application files
tar -czf $BACKUP_DIR/sendense-gui_$DATE.tar.gz \
    -C /opt sendense-gui \
    --exclude=node_modules \
    --exclude=.next/cache

# Keep only last 7 backups
cd $BACKUP_DIR
ls -t *.tar.gz | tail -n +8 | xargs -r rm
```

## 🔄 Updates and Rollbacks

### **Zero-Downtime Updates**
```bash
# Stop current service
sudo systemctl stop sendense-gui

# Backup current version
cp -r /opt/sendense-gui /opt/sendense-gui.backup.$(date +%s)

# Deploy new version
cd /home/oma_admin/sendense/source/current/sendense-gui
npm run build
cp -r .next/* /opt/sendense-gui/.next/

# Start service
sudo systemctl start sendense-gui

# Verify
curl -f http://localhost:3001

# Remove backup after 24h if successful
# find /opt -name "sendense-gui.backup.*" -mtime +1 -exec rm -rf {} \;
```

### **Rollback**
```bash
# Stop service
sudo systemctl stop sendense-gui

# Restore backup
BACKUP=$(ls -t /opt/sendense-gui.backup.* | head -1)
cp -r $BACKUP/* /opt/sendense-gui/

# Start service
sudo systemctl start sendense-gui
```

## 🚨 Troubleshooting

See `GUI-TROUBLESHOOTING.md` for common issues and solutions.

## 📈 Performance Optimization

### **Post-Deployment Verification**
```bash
# Lighthouse performance test
npx lighthouse http://localhost:3001 --output=json --output-path=./report.json

# Bundle analyzer
npx @next/bundle-analyzer
```

### **Expected Performance**
- **First Contentful Paint**: < 1.5s
- **Largest Contentful Paint**: < 2.5s
- **Cumulative Layout Shift**: < 0.1
- **Bundle Size**: < 200KB per page

---

**Deployment Complete**: GUI is now live at `http://your-server:3001` or configured domain.
