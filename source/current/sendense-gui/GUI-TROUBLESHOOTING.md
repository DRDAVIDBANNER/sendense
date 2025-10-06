# Sendense GUI Troubleshooting Guide

## 🔍 Common Issues and Solutions

This guide covers troubleshooting production deployment issues for the Sendense GUI.

## 🚫 Build Failures

### **TypeScript Errors During Build**

#### **Issue**: `npm run build` fails with TypeScript errors
```
Failed to compile.
./app/some-file.tsx:123:45
Type error: ...
```

#### **Solutions**:
1. **Check TypeScript Configuration**:
   ```bash
   # Verify tsconfig.json
   cat tsconfig.json | jq '.compilerOptions.strict'

   # Should be: "strict": true
   ```

2. **Clear Cache and Rebuild**:
   ```bash
   rm -rf .next node_modules/.cache
   npm ci
   npm run build
   ```

3. **Check Import Paths**:
   ```bash
   # Verify all @ imports resolve
   grep -r "@/components" app/ --include="*.tsx" | head -5
   ```

4. **Type Definition Issues**:
   - Ensure `types.ts` files exist in component directories
   - Check Flow types are properly exported

### **Missing Page Files**

#### **Issue**: `Cannot find module for page: /some-route`
```
[Error [PageNotFoundError]: Cannot find module for page: /dashboard]
```

#### **Solutions**:
1. **Verify File Structure**:
   ```bash
   find app -name "page.tsx" | sort
   # Should list all expected pages
   ```

2. **Check File Permissions**:
   ```bash
   ls -la app/dashboard/page.tsx
   # Should be readable by build process
   ```

3. **Clear Next.js Cache**:
   ```bash
   rm -rf .next
   npm run build
   ```

## 🚫 Runtime Errors

### **Application Won't Start**

#### **Issue**: `npm run start` fails or exits immediately

#### **Solutions**:
1. **Check Port Availability**:
   ```bash
   netstat -tlnp | grep :3000
   # Kill conflicting process if needed
   sudo kill -9 <PID>
   ```

2. **Verify Build Artifacts**:
   ```bash
   ls -la .next/
   # Should contain server/ and static/ directories
   ```

3. **Check Node.js Version**:
   ```bash
   node --version
   # Should be 18+ (tested with 19.1.0)
   ```

4. **Environment Variables**:
   ```bash
   # Check required env vars
   echo $NODE_ENV
   # Should be 'production'
   ```

### **White Screen or 404 Errors**

#### **Issue**: Application starts but shows blank page or 404

#### **Solutions**:
1. **Check Browser Console**:
   - Open Developer Tools → Console
   - Look for JavaScript errors
   - Check network tab for failed asset loads

2. **Verify Static Assets**:
   ```bash
   ls -la .next/static/
   # Should contain CSS and JS chunks
   ```

3. **Base Path Issues**:
   ```bash
   # Check next.config.ts for basePath
   grep -n "basePath" next.config.ts
   ```

4. **CORS Issues**:
   - Check API endpoints are accessible
   - Verify environment variables for API URLs

## 🚫 Performance Issues

### **Slow Page Loads**

#### **Issue**: Pages load slowly (>3 seconds)

#### **Solutions**:
1. **Check Bundle Sizes**:
   ```bash
   npm run build
   # Review bundle size output
   # Largest page should be <200KB
   ```

2. **Enable Compression**:
   ```bash
   # Verify gzip in next.config.ts
   grep -n "compress" next.config.ts
   ```

3. **Static Asset Caching**:
   ```bash
   # Check nginx config for static caching
   grep -A5 "_next/static" /etc/nginx/sites-available/*
   ```

4. **Database Queries**:
   - Check for slow API calls in Network tab
   - Optimize database queries if applicable

### **Memory Issues**

#### **Issue**: Application crashes with out of memory errors

#### **Solutions**:
1. **Increase Node.js Memory**:
   ```bash
   # Update systemd service
   sudo sed -i 's|ExecStart=/usr/bin/npm run start|ExecStart=/usr/bin/node --max-old-space-size=2048 /usr/bin/npm run start|' /etc/systemd/system/sendense-gui.service
   sudo systemctl daemon-reload
   sudo systemctl restart sendense-gui
   ```

2. **Monitor Memory Usage**:
   ```bash
   # Check current usage
   ps aux | grep node
   ```

3. **Optimize Components**:
   - Use React.memo for expensive components
   - Implement proper lazy loading
   - Check for memory leaks in useEffect

## 🚫 Networking Issues

### **Cannot Access Application**

#### **Issue**: Application runs but isn't accessible externally

#### **Solutions**:
1. **Check Service Status**:
   ```bash
   sudo systemctl status sendense-gui
   sudo journalctl -u sendense-gui -n 20
   ```

2. **Verify Port Binding**:
   ```bash
   ss -tlnp | grep :3001
   # Should show node process listening
   ```

3. **Firewall Rules**:
   ```bash
   sudo ufw status
   # Allow port 3001 if needed
   sudo ufw allow 3001
   ```

4. **Reverse Proxy Issues**:
   ```bash
   # Test direct access
   curl http://localhost:3001

   # Check nginx config
   sudo nginx -t
   sudo systemctl reload nginx
   ```

### **SSL/TLS Issues**

#### **Issue**: HTTPS not working or certificate errors

#### **Solutions**:
1. **Certificate Validity**:
   ```bash
   openssl x509 -in /path/to/cert.pem -text -noout | grep -A2 "Validity"
   ```

2. **nginx SSL Configuration**:
   ```bash
   sudo nginx -t
   sudo systemctl reload nginx
   ```

3. **Mixed Content**:
   - Ensure all assets load over HTTPS
   - Check API calls use HTTPS URLs

## 🚫 Database Issues

### **API Calls Failing**

#### **Issue**: Frontend loads but API requests fail

#### **Solutions**:
1. **Check API Endpoints**:
   ```bash
   curl -f https://api.sendense.com/health
   ```

2. **Environment Variables**:
   ```bash
   # Verify API URLs in .env
   cat .env.production | grep API_URL
   ```

3. **CORS Configuration**:
   - Check API server allows GUI domain
   - Verify CORS headers in API responses

4. **Network Connectivity**:
   ```bash
   traceroute api.sendense.com
   telnet api.sendense.com 443
   ```

## 🔧 Development Mode Issues

### **Development Server Not Starting**

#### **Issue**: `npm run dev` fails

#### **Solutions**:
1. **Port Conflicts**:
   ```bash
   lsof -i :3000
   # Kill conflicting process
   ```

2. **Dependencies**:
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

3. **Cache Issues**:
   ```bash
   rm -rf .next
   npm run dev
   ```

## 📊 Diagnostic Commands

### **System Diagnostics**
```bash
# System resources
df -h
free -h
uptime

# Service status
sudo systemctl status sendense-gui
sudo journalctl -u sendense-gui --since "1 hour ago"

# Network
ss -tlnp | grep :3001
curl -I http://localhost:3001
```

### **Application Diagnostics**
```bash
# Build verification
npm run build 2>&1 | tail -20

# Dependency check
npm ls --depth=0

# Environment check
env | grep -E "(NODE|NEXT)"

# Log analysis
tail -f /opt/sendense-gui/logs/application.log
```

### **Performance Diagnostics**
```bash
# Lighthouse audit
npx lighthouse http://localhost:3001 --output=json

# Bundle analyzer
npm install --save-dev @next/bundle-analyzer
npx @next/bundle-analyzer

# Memory profiling
node --inspect --max-old-space-size=2048 node_modules/.bin/next start
```

## 🚨 Emergency Recovery

### **Quick Restart**
```bash
sudo systemctl restart sendense-gui
sleep 5
curl -f http://localhost:3001
```

### **Full Recovery**
```bash
# Stop service
sudo systemctl stop sendense-gui

# Clear caches
rm -rf /opt/sendense-gui/.next

# Rebuild application
cd /home/oma_admin/sendense/source/current/sendense-gui
npm run build
cp -r .next/* /opt/sendense-gui/.next/

# Restart service
sudo systemctl start sendense-gui
```

### **Rollback to Previous Version**
```bash
# Find latest backup
ls -la /opt/sendense-gui.backup.*

# Restore
sudo systemctl stop sendense-gui
cp -r /opt/sendense-gui.backup.*/.next /opt/sendense-gui/
sudo systemctl start sendense-gui
```

## 📞 Getting Help

If issues persist:

1. **Check Logs**: `sudo journalctl -u sendense-gui -f`
2. **Gather Diagnostics**: Run diagnostic commands above
3. **Document Issue**: Include error messages, steps to reproduce
4. **Contact Support**: Provide system info and log excerpts

---

**Remember**: Always test changes in development before deploying to production!
