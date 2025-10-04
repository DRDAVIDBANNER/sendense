# VMA Deployment Troubleshooting

## Common Issues

### 1. Config Syntax Errors
**Symptom**: Service fails with "command not found" error
**Cause**: Unquoted SETUP_DATE in config file
**Fix**: Use quoted date in config template

### 2. SSH Tunnel Authentication
**Symptom**: Tunnel service restarts constantly
**Cause**: SSH key not deployed or wrong permissions
**Fix**: Verify keys in /home/vma/.ssh/ with proper permissions

### 3. Service Startup Failures
**Symptom**: Services fail to start after deployment
**Cause**: Missing dependencies or wrong binary paths
**Fix**: Check service logs with journalctl -u <service>

## Validation Commands
```bash
# Service status
systemctl status vma-api vma-ssh-tunnel

# Binary verification
test -x /opt/vma/bin/migratekit && echo "MigrateKit OK"
test -x /opt/vma/bin/vma-api-server && echo "VMA API OK"

# SSH key verification
test -f /home/vma/.ssh/cloudstack_key && echo "SSH keys OK"
```
