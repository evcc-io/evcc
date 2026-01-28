#!/bin/bash
# EcoFlow BatteryController Test Script
#
# Usage:
#   export ECOFLOW_SN="your-serial-number"
#   export ECOFLOW_ACCESS_KEY="your-access-key"
#   export ECOFLOW_SECRET_KEY="your-secret-key"
#   ./ecoflow-control.sh status
#   ./ecoflow-control.sh normal
#   ./ecoflow-control.sh hold
#   ./ecoflow-control.sh charge

set -e

# Check required environment variables
if [ -z "$ECOFLOW_SN" ] || [ -z "$ECOFLOW_ACCESS_KEY" ] || [ -z "$ECOFLOW_SECRET_KEY" ]; then
    echo "❌ Missing required environment variables:"
    echo "   export ECOFLOW_SN='your-serial-number'"
    echo "   export ECOFLOW_ACCESS_KEY='your-access-key'"
    echo "   export ECOFLOW_SECRET_KEY='your-secret-key'"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

cd "$REPO_DIR"

CMD="${1:-status}"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "EcoFlow BatteryController Test"
echo "Device: $ECOFLOW_SN"
echo "Command: $CMD"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

go run "$SCRIPT_DIR/test_battery_control.go" "$CMD"
