# Sendense Hub Appliance (SHA) Binaries

## Current Production Binaries

### Sendense Hub API Server
- **Binary:** `sendense-hub-latest` → `sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d`
- **Version:** v2.8.1-nbd-progress-tracking
- **Commit:** 2cf590d
- **Built:** 2025-10-05
- **Size:** 33 MB

### Volume Management Daemon
- **Binary:** `volume-daemon-latest` → `volume-daemon-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d`
- **Version:** v2.8.1-nbd-progress-tracking
- **Commit:** 2cf590d
- **Built:** 2025-10-05
- **Size:** 2.2 MB

## Binary Naming Convention

Format: `{component}-v{version}-{platform}-{arch}-{date}-{commit}`

**Example:**
```
sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
```

Components:
- `{component}`: sendense-hub, volume-daemon
- `{version}`: Semantic version from source/current/VERSION.txt
- `{platform}`: linux, darwin, windows
- `{arch}`: amd64, arm64
- `{date}`: YYYYMMDD build date
- `{commit}`: Short git commit hash (7 chars)

## Building New Binaries

### Prerequisites
```bash
# Ensure you're in the sendense workspace
cd /home/oma_admin/sendense
```

### Build Sendense Hub
```bash
cd source/current/oma
VERSION=$(cat ../VERSION.txt)
COMMIT=$(git log --oneline -1 | awk '{print $1}')
DATE=$(date +%Y%m%d)
BINARY_NAME="sendense-hub-${VERSION}-linux-amd64-${DATE}-${COMMIT}"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o "$BINARY_NAME" ./cmd/main.go

# Verify
ls -lh "$BINARY_NAME"
file "$BINARY_NAME"
```

### Build Volume Daemon
```bash
cd source/current/volume-daemon
VERSION=$(cat ../VERSION.txt)
COMMIT=$(git log --oneline -1 | awk '{print $1}')
DATE=$(date +%Y%m%d)
BINARY_NAME="volume-daemon-${VERSION}-linux-amd64-${DATE}-${COMMIT}"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o "$BINARY_NAME" .

# Verify
ls -lh "$BINARY_NAME"
file "$BINARY_NAME"
```

## Deploying New Binaries

### 1. Copy binaries to deployment directory
```bash
cd /home/oma_admin/sendense/deployment/sha-appliance/binaries

# Copy new binaries
cp ../../source/current/oma/sendense-hub-v* .
cp ../../source/current/volume-daemon/volume-daemon-v* .

# Make executable
chmod +x sendense-hub-v*
chmod +x volume-daemon-v*
```

### 2. Update symlinks
```bash
# Remove old symlinks
rm -f sendense-hub-latest volume-daemon-latest

# Create new symlinks pointing to new binaries
ln -s sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d sendense-hub-latest
ln -s volume-daemon-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d volume-daemon-latest

# Verify
ls -l *-latest
```

### 3. Generate checksums
```bash
# Individual checksums
sha256sum sendense-hub-v2.8.1* > sendense-hub-v2.8.1.sha256
sha256sum volume-daemon-v2.8.1* > volume-daemon-v2.8.1.sha256

# Combined checksums for verification
sha256sum sendense-hub-v2.8.1* volume-daemon-v2.8.1* > CHECKSUMS-v2.8.1.sha256
```

### 4. Deploy to remote server
```bash
cd /home/oma_admin/sendense/deployment/sha-appliance/scripts

# The deploy script automatically uses the *-latest symlinks
./deploy-sha-remote.sh <TARGET_IP>
```

## Quick Update Process

Once you've built new binaries, this is all you need:

```bash
cd /home/oma_admin/sendense/deployment/sha-appliance/binaries

# Copy new binaries
cp ../../source/current/oma/sendense-hub-v* .
cp ../../source/current/volume-daemon/volume-daemon-v* .
chmod +x sendense-hub-v* volume-daemon-v*

# Update symlinks to point to new versions
rm sendense-hub-latest volume-daemon-latest
ln -s sendense-hub-v<NEW_VERSION>-linux-amd64-<DATE>-<COMMIT> sendense-hub-latest
ln -s volume-daemon-v<NEW_VERSION>-linux-amd64-<DATE>-<COMMIT> volume-daemon-latest

# Generate checksums
sha256sum sendense-hub-v* volume-daemon-v* > CHECKSUMS-latest.sha256

# Deploy
cd ../scripts
./deploy-sha-remote.sh <TARGET_IP>
```

## Verifying Deployed Binaries

On the remote server after deployment:

```bash
# Check versions
/opt/sendense/bin/sendense-hub --version
/usr/local/bin/volume-daemon --version

# Check service status
systemctl status sendense-hub
systemctl status volume-daemon

# Check API health
curl http://localhost:8082/health
curl http://localhost:8090/api/v1/health
```

## Binary Management Best Practices

1. **Always build from clean source**
   ```bash
   cd source/current && git status  # Check for uncommitted changes
   ```

2. **Always include version info in ldflags**
   - Enables `--version` flag
   - Shows version in logs
   - Helps troubleshooting

3. **Use symlinks for deployment**
   - Deploy script uses `sendense-hub-latest` and `volume-daemon-latest`
   - Easy to roll back: just update symlink to previous version
   - No need to modify deployment scripts

4. **Keep old binaries for rollback**
   - Keep at least last 3 versions
   - Store with full version info in filename
   - Document what changed between versions

5. **Generate checksums for verification**
   - Ensures binary integrity during transfer
   - Verifies no corruption occurred
   - Required for security compliance

## Version History

### v2.8.1-nbd-progress-tracking (2025-10-05)
- **Commit:** 2cf590d
- **Changes:** Initial backup export helpers test suite (Task 2.3 prep)
- **Sendense Hub:** 33 MB
- **Volume Daemon:** 2.2 MB
- **Status:** Production deployment

### v2.7.6-api-uuid-correlation (2025-10-05)
- **Previous version** - Replaced
- **Status:** Archived

## Troubleshooting

### Binary won't execute
```bash
# Check if it's actually an ELF binary
file sendense-hub-latest

# Check permissions
ls -l sendense-hub-latest

# Make executable
chmod +x sendense-hub-latest
```

### Version flag doesn't work
- Ensure ldflags were included during build
- Rebuild with proper ldflags

### Service fails to start
```bash
# Check service logs
journalctl -u sendense-hub -n 50
journalctl -u volume-daemon -n 50

# Check binary can run
/opt/sendense/bin/sendense-hub --version
```

## Contact

For build issues or questions about binaries, refer to:
- `source/current/VERSION.txt` - Current version
- `start_here/PROJECT_RULES.md` - Build standards
- `deployment/sha-appliance/scripts/` - Deployment scripts
