# VMA Configuration Template Usage

## Template File: vma-config.conf.template

### Variable Substitution:
- __OMA_IP_PLACEHOLDER__ → Target OMA IP address
- __SETUP_DATE_PLACEHOLDER__ → Current date/time (quoted)

### Usage in Deployment Script:
```bash
sed 's/__OMA_IP_PLACEHOLDER__/10.245.246.147/g; s/__SETUP_DATE_PLACEHOLDER__/'"$(date)"'/g' vma-config.conf.template > /opt/vma/vma-config.conf
```

### Usage in Wizard:
Template prevents syntax errors from unquoted date expansion.
