# VMA-OMA Integration Procedures

## SSH Tunnel Setup
1. **VMA Side**: Deploy VMA with pre-shared key
2. **OMA Side**: Add VMA public key to vma_tunnel user
3. **Connection**: VMA connects to vma_tunnel@OMA:443

## Tunnel Validation
```bash
# On VMA: Test tunnel ports
ss -tlnp | grep -E ":10808|:8082"

# On OMA: Test reverse tunnel
ss -tlnp | grep :9081
```

## Migration Workflow
1. VMA discovers VMware VMs
2. OMA creates migration jobs
3. VMA executes migration via NBD tunnel
4. Progress reported via OMA API tunnel
