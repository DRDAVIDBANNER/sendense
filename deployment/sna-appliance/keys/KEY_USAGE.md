# VMA SSH Key Management

## Pre-shared Key: cloudstack_key
- **Private Key**: cloudstack_key (600 permissions)
- **Public Key**: cloudstack_key.pub (644 permissions)
- **Usage**: VMA tunnel authentication to OMA
- **Security**: RSA 2048-bit key pair

## Deployment Usage:
1. Private key deployed to VMA: `/home/vma/.ssh/cloudstack_key`
2. Public key deployed to OMA: `/var/lib/vma_tunnel/.ssh/authorized_keys`

## Security Requirements:
- Private key must have 600 permissions
- Private key must be owned by vma:vma on VMA
- Public key must be in vma_tunnel authorized_keys on OMA
