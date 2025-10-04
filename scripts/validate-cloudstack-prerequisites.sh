#!/bin/bash
# CloudStack Prerequisite Validation Script
# 
# Purpose: Validate CloudStack configuration before deployment/migration
# Usage: ./validate-cloudstack-prerequisites.sh [--auto-fix] [--config-name NAME]
#
# Created: October 3, 2025

set -e

# Configuration
OMA_API_URL="${OMA_API_URL:-http://localhost:8082}"
CONFIG_NAME="${CONFIG_NAME:-production-ossea}"
AUTO_FIX="${AUTO_FIX:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --auto-fix)
            AUTO_FIX="true"
            shift
            ;;
        --config-name)
            CONFIG_NAME="$2"
            shift 2
            ;;
        --api-url)
            OMA_API_URL="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --auto-fix          Attempt to automatically fix issues"
            echo "  --config-name NAME  CloudStack config name (default: production-ossea)"
            echo "  --api-url URL       OMA API URL (default: http://localhost:8082)"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}🔍 CloudStack Prerequisite Validation${NC}"
echo "==========================================="
echo "Config: $CONFIG_NAME"
echo "Auto-fix: $AUTO_FIX"
echo "API URL: $OMA_API_URL"
echo ""

# Check if OMA API is accessible
echo -e "${BLUE}Checking OMA API accessibility...${NC}"
if ! curl -s -f "$OMA_API_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Cannot connect to OMA API at $OMA_API_URL${NC}"
    echo "Make sure OMA API is running and accessible"
    exit 1
fi
echo -e "${GREEN}✅ OMA API is accessible${NC}"
echo ""

# Run validation
echo -e "${BLUE}Running CloudStack prerequisite validation...${NC}"
VALIDATION_REQUEST=$(cat <<EOF
{
  "config_name": "$CONFIG_NAME",
  "auto_fix": $AUTO_FIX
}
EOF
)

VALIDATION_RESULT=$(curl -s -X POST "$OMA_API_URL/api/v1/cloudstack/validate" \
  -H "Content-Type: application/json" \
  -d "$VALIDATION_REQUEST")

# Check if validation was successful
SUCCESS=$(echo "$VALIDATION_RESULT" | jq -r '.success')
OVERALL_PASSED=$(echo "$VALIDATION_RESULT" | jq -r '.validation_report.overall_passed')
TOTAL_CHECKS=$(echo "$VALIDATION_RESULT" | jq -r '.validation_report.total_checks')
PASSED_CHECKS=$(echo "$VALIDATION_RESULT" | jq -r '.validation_report.passed_checks')
FAILED_CHECKS=$(echo "$VALIDATION_RESULT" | jq -r '.validation_report.failed_checks')
CRITICAL_FAILURES=$(echo "$VALIDATION_RESULT" | jq -r '.validation_report.critical_failures')

echo ""
echo "==========================================="
echo -e "${BLUE}Validation Results:${NC}"
echo "Total checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo -e "${RED}Critical failures: $CRITICAL_FAILURES${NC}"
echo "==========================================="
echo ""

# Show failed checks
if [ "$FAILED_CHECKS" -gt "0" ]; then
    echo -e "${RED}❌ Failed Checks:${NC}"
    echo "$VALIDATION_RESULT" | jq -r '.validation_report.results[] | select(.passed == false) | 
        "[\(.severity)] \(.category) - \(.check_name):\n  Message: \(.message)\n  Fix: \(.fix)\n"'
fi

# Show auto-fix results if auto-fix was enabled
if [ "$AUTO_FIX" = "true" ]; then
    FIXES_ATTEMPTED=$(echo "$VALIDATION_RESULT" | jq -r '.auto_fix_report.fixes_attempted // 0')
    FIXES_SUCCESSFUL=$(echo "$VALIDATION_RESULT" | jq -r '.auto_fix_report.fixes_successful // 0')
    FIXES_FAILED=$(echo "$VALIDATION_RESULT" | jq -r '.auto_fix_report.fixes_failed // 0')
    
    if [ "$FIXES_ATTEMPTED" -gt "0" ]; then
        echo ""
        echo "==========================================="
        echo -e "${BLUE}Auto-Fix Results:${NC}"
        echo "Attempted: $FIXES_ATTEMPTED"
        echo -e "${GREEN}Successful: $FIXES_SUCCESSFUL${NC}"
        echo -e "${RED}Failed: $FIXES_FAILED${NC}"
        echo "==========================================="
        echo ""
        
        if [ "$FIXES_SUCCESSFUL" -gt "0" ]; then
            echo -e "${GREEN}✅ Auto-Fix Results:${NC}"
            echo "$VALIDATION_RESULT" | jq -r '.auto_fix_report.results[] | select(.successful == true) | 
                "✅ \(.fix_name): \(.message)"'
            echo ""
        fi
        
        if [ "$FIXES_FAILED" -gt "0" ]; then
            echo -e "${RED}❌ Auto-Fix Failures:${NC}"
            echo "$VALIDATION_RESULT" | jq -r '.auto_fix_report.results[] | select(.successful == false) | 
                "❌ \(.fix_name): \(.message)\n  Details: \(.details)"'
            echo ""
        fi
    fi
fi

# Show summary and exit with appropriate code
echo ""
echo "==========================================="
if [ "$OVERALL_PASSED" = "true" ] && [ "$CRITICAL_FAILURES" -eq "0" ]; then
    echo -e "${GREEN}✅ VALIDATION PASSED${NC}"
    echo "All critical CloudStack prerequisites are met."
    echo "You can proceed with deployment/migration."
    echo "==========================================="
    exit 0
else
    echo -e "${RED}❌ VALIDATION FAILED${NC}"
    echo "$CRITICAL_FAILURES critical issue(s) must be resolved before proceeding."
    echo ""
    if [ "$AUTO_FIX" = "false" ]; then
        echo "💡 Tip: Run with --auto-fix to attempt automatic fixes"
    fi
    echo "==========================================="
    exit 1
fi


