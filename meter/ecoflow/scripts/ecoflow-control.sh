#!/bin/bash
# EcoFlow BatteryController Test Script
#
# Usage:
#   ./ecoflow-control.sh status
#   ./ecoflow-control.sh normal
#   ./ecoflow-control.sh hold
#   ./ecoflow-control.sh charge

set -e

# Default credentials (override with env vars)
export ECOFLOW_SN="${ECOFLOW_SN:-BK61ZE1B2H6H0912}"
export ECOFLOW_ACCESS_KEY="${ECOFLOW_ACCESS_KEY:-Ms0Nefw3xBOHZMA36l8fD7IzXteWLvLL}"
export ECOFLOW_SECRET_KEY="${ECOFLOW_SECRET_KEY:-uzDj9L9F5v5DFGObypJH5vlAcHkNPYn8}"

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
