#!/bin/bash
# EcoFlow Integration Test Runner
#
# Usage:
#   ./run_integration_test.sh [options]
#
# Options:
#   --status      Run full status report only
#   --read        Run read tests only
#   --control     Run control tests (requires ECOFLOW_ALLOW_CONTROL=true)
#   --all         Run all tests
#
# Environment variables (required):
#   ECOFLOW_SN          Device serial number
#   ECOFLOW_ACCESS_KEY  API access key  
#   ECOFLOW_SECRET_KEY  API secret key
#
# Optional:
#   ECOFLOW_URI         API URI (default: https://api-e.ecoflow.com)
#   ECOFLOW_DEVICE      Device type: "stream" or "powerstream" (default: stream)
#   ECOFLOW_ALLOW_CONTROL Set to "true" to enable control tests

set -e
cd "$(dirname "$0")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check credentials
if [[ -z "$ECOFLOW_SN" || -z "$ECOFLOW_ACCESS_KEY" || -z "$ECOFLOW_SECRET_KEY" ]]; then
    echo -e "${RED}Error: Missing credentials${NC}"
    echo ""
    echo "Required environment variables:"
    echo "  export ECOFLOW_SN='YOUR_SERIAL_NUMBER'"
    echo "  export ECOFLOW_ACCESS_KEY='YOUR_ACCESS_KEY'"
    echo "  export ECOFLOW_SECRET_KEY='YOUR_SECRET_KEY'"
    echo ""
    echo "Optional:"
    echo "  export ECOFLOW_DEVICE='stream'  # or 'powerstream'"
    echo "  export ECOFLOW_ALLOW_CONTROL='true'  # enable control tests"
    exit 1
fi

echo -e "${GREEN}EcoFlow Integration Tests${NC}"
echo "================================"
echo "Device SN: ${ECOFLOW_SN}"
echo "Device Type: ${ECOFLOW_DEVICE:-stream}"
echo "Control Tests: ${ECOFLOW_ALLOW_CONTROL:-disabled}"
echo ""

case "${1:-all}" in
    --status)
        echo -e "${YELLOW}Running status report...${NC}"
        go test -tags=integration -v -run TestIntegration_FullStatus ./...
        ;;
    --read)
        echo -e "${YELLOW}Running read tests...${NC}"
        go test -tags=integration -v -run "TestIntegration_Read" ./...
        ;;
    --control)
        if [[ "$ECOFLOW_ALLOW_CONTROL" != "true" ]]; then
            echo -e "${RED}Control tests require ECOFLOW_ALLOW_CONTROL=true${NC}"
            exit 1
        fi
        echo -e "${YELLOW}Running control tests...${NC}"
        go test -tags=integration -v -run "TestIntegration_Control" ./...
        ;;
    --all|*)
        echo -e "${YELLOW}Running all tests...${NC}"
        go test -tags=integration -v ./...
        ;;
esac

echo ""
echo -e "${GREEN}✅ Tests completed${NC}"
