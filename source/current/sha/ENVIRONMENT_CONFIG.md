# OMA API Environment Configuration

## VMA Power Management Configuration

The OMA API now supports secure environment-based configuration for vCenter credentials used in VMA power management operations.

### Required Environment Variables

For production deployment, set the following environment variables:

```bash
# VMA Tunnel Configuration
export VMA_TUNNEL_ENDPOINT="http://localhost:9081"

# vCenter Connection Details
export VCENTER_HOST="192.168.17.159"
export VCENTER_USERNAME="administrator@vsphere.local"
export VCENTER_PASSWORD="EmyGVoBFesGQc47-"
```

### Environment Variable Details

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `VMA_TUNNEL_ENDPOINT` | VMA tunnel endpoint URL | `http://localhost:9081` | No |
| `VCENTER_HOST` | vCenter server hostname or IP | None | **Yes** |
| `VCENTER_USERNAME` | vCenter authentication username | None | **Yes** |
| `VCENTER_PASSWORD` | vCenter authentication password | None | **Yes** |

### Deployment Examples

#### Development Environment
```bash
# Set environment variables
export VCENTER_HOST="192.168.17.159"
export VCENTER_USERNAME="administrator@vsphere.local"
export VCENTER_PASSWORD="EmyGVoBFesGQc47-"

# Start OMA API
./oma-api-secure-vma-power
```

#### Production Systemd Service
```ini
[Unit]
Description=OMA API with VMA Power Management
After=network.target

[Service]
Type=simple
User=oma
Environment=VMA_TUNNEL_ENDPOINT=http://localhost:9081
Environment=VCENTER_HOST=your-vcenter-host
Environment=VCENTER_USERNAME=your-vcenter-user
Environment=VCENTER_PASSWORD=your-vcenter-password
ExecStart=/opt/oma/oma-api-secure-vma-power
Restart=always

[Install]
WantedBy=multi-user.target
```

#### Docker Deployment
```bash
docker run -d \
  -e VCENTER_HOST="192.168.17.159" \
  -e VCENTER_USERNAME="administrator@vsphere.local" \
  -e VCENTER_PASSWORD="EmyGVoBFesGQc47-" \
  -p 8080:8080 \
  oma-api-secure-vma-power
```

### Fallback Behavior

If environment variables are not set, the system will:
1. **Try environment configuration first** via `NewVMAClientFromConfig()`
2. **Fallback to known working configuration** via `NewVMAClientWithDefaults()`
3. **Disable power management** if all configuration fails (uses `NullVMAClient`)

**⚠️ Security Note**: The fallback configuration should be removed in production environments by replacing `NewVMAClientWithDefaults()` with `NewVMAClientFromConfig()` and handling errors appropriately.

### Verification

To verify environment configuration is working:

```bash
# Check environment variables are set
env | grep -E "(VMA_|VCENTER_)"

# Start OMA API and check logs for VMA client initialization
./oma-api-secure-vma-power | grep -i "vma client"
```

### Security Best Practices

1. **Never commit credentials** to source code
2. **Use environment variables** for all sensitive configuration
3. **Rotate credentials regularly** in production
4. **Use secure credential management** systems in production environments
5. **Limit vCenter user permissions** to minimum required for power management

### Troubleshooting

**VMA Client Initialization Fails**:
- Check all required environment variables are set
- Verify vCenter credentials are correct
- Ensure VMA tunnel is accessible at specified endpoint
- Check OMA API logs for specific error messages

**Power Management Operations Fail**:
- Verify VMA appliance is running and accessible
- Check vCenter connectivity from VMA appliance
- Validate VMware VM IDs are correct
- Review VMA API logs for VMware-specific errors


