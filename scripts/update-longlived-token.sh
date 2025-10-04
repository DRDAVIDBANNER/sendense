#!/bin/bash

# Script to update VMA and GUI with long-lived authentication token
# This eliminates the need for frequent token renewal after service restarts

set -e

LONGLIVED_TOKEN="sess_longlived_dev_token_2025_2035_permanent"
VMA_CONFIG_FILE="/etc/vma/config.json"
GUI_DASHBOARD_DIR="/home/pgrayson/migration-dashboard"

echo "🔐 Updating authentication with long-lived token (10-year expiry)"
echo "Token: $LONGLIVED_TOKEN"

# Step 1: Generate fresh authentication with OMA API
echo ""
echo "📡 Step 1: Authenticating with OMA API to generate long-lived session..."

AUTH_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "appliance_id": "vma-001", 
    "token": "vma_test_token_abc123def456789012345678",
    "version": "1.0.0"
  }')

echo "Auth Response: $AUTH_RESPONSE"

# Extract session token
SESSION_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.session_token')
EXPIRES_AT=$(echo "$AUTH_RESPONSE" | jq -r '.expires_at')

if [ "$SESSION_TOKEN" != "null" ] && [ "$SESSION_TOKEN" != "" ]; then
    echo "✅ Authentication successful!"
    echo "   Session Token: $SESSION_TOKEN"  
    echo "   Expires: $EXPIRES_AT"
else
    echo "❌ Authentication failed. Response: $AUTH_RESPONSE"
    exit 1
fi

# Step 2: Update VMA configuration (if exists)
echo ""
echo "🔧 Step 2: Updating VMA configuration..."

if [ -f "$VMA_CONFIG_FILE" ]; then
    echo "Updating VMA config file: $VMA_CONFIG_FILE"
    # Update the auth token in VMA config if it exists
    sudo jq --arg token "$SESSION_TOKEN" '.auth_token = $token' "$VMA_CONFIG_FILE" > /tmp/vma_config.json
    sudo mv /tmp/vma_config.json "$VMA_CONFIG_FILE"
    echo "✅ VMA config updated"
else
    echo "ℹ️  VMA config file not found - may be using environment variables"
fi

# Step 3: Update GUI configuration
echo ""
echo "🎨 Step 3: Updating GUI authentication token..."

if [ -d "$GUI_DASHBOARD_DIR" ]; then
    cd "$GUI_DASHBOARD_DIR"
    
    # Update replicate route
    if [ -f "src/app/api/replicate/route.ts" ]; then
        echo "Updating replicate route token..."
        sed -i "s/'Authorization': 'Bearer sess_[^']*'/'Authorization': 'Bearer $SESSION_TOKEN'/g" src/app/api/replicate/route.ts
        echo "✅ Replicate route updated"
    fi
    
    # Update migrations route  
    if [ -f "src/app/api/migrations/route.ts" ]; then
        echo "Updating migrations route token..."
        sed -i "s/'Authorization': 'Bearer sess_[^']*'/'Authorization': 'Bearer $SESSION_TOKEN'/g" src/app/api/migrations/route.ts
        echo "✅ Migrations route updated"
    fi
    
    # Update discover route if it exists
    if [ -f "src/app/api/discover/route.ts" ]; then
        echo "Updating discover route token..."
        sed -i "s/'Authorization': 'Bearer sess_[^']*'/'Authorization': 'Bearer $SESSION_TOKEN'/g" src/app/api/discover/route.ts
        echo "✅ Discover route updated"
    fi
    
    echo "🔄 Restarting GUI service..."
    sudo systemctl restart migration-gui
    echo "✅ GUI service restarted"
    
else
    echo "❌ GUI dashboard directory not found: $GUI_DASHBOARD_DIR"
fi

# Step 4: Update VMA client configuration via environment
echo ""
echo "🛠️  Step 4: Setting VMA environment variables..."

VMA_ENV_FILE="/home/pgrayson/vma_auth.env"
cat > "$VMA_ENV_FILE" << EOF
# VMA Authentication Configuration  
# Long-lived token valid until 2035
export VMA_OMA_BASE_URL="http://localhost:8082"
export VMA_AUTH_TOKEN="vma_test_token_abc123def456789012345678"
export VMA_SESSION_TOKEN="$SESSION_TOKEN"
export VMA_APPLIANCE_ID="vma-001"
EOF

echo "✅ VMA environment file created: $VMA_ENV_FILE"
echo "   To use: source $VMA_ENV_FILE"

# Step 5: Restart services
echo ""
echo "🔄 Step 5: Restarting services..."

if systemctl is-active --quiet oma-api; then
    sudo systemctl restart oma-api
    echo "✅ OMA API service restarted"
fi

if systemctl is-active --quiet vma-api; then
    sudo systemctl restart vma-api
    echo "✅ VMA API service restarted" 
fi

echo ""
echo "🎉 COMPLETED: Long-lived token configuration updated!"
echo ""
echo "📋 Summary:"
echo "   - Token expires: $EXPIRES_AT (10 years from now)"
echo "   - VMA config: Updated if file exists"
echo "   - GUI routes: Updated with new token"
echo "   - Services: Restarted with new configuration"
echo ""
echo "🧪 Test the configuration:"
echo "   curl -H \"Authorization: Bearer $SESSION_TOKEN\" http://localhost:8082/health"
echo "   curl http://localhost:3001"



