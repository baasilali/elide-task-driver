#!/bin/bash
# Copyright (c) Elide Dev, Inc.
# SPDX-License-Identifier: Apache-2.0

# End-to-end test script for Elide task driver
# This script tests the complete flow: server -> Nomad -> driver -> execution

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=========================================="
echo "Elide Task Driver - End-to-End Test"
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
command -v nomad >/dev/null 2>&1 || { echo -e "${RED}nomad is required but not installed.${NC}" >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo -e "${RED}go is required but not installed.${NC}" >&2; exit 1; }

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    pkill -f "nomad agent" || true
    pkill -f "cmd/server/main.go" || true
    rm -f /tmp/elide-daemon.sock
    echo -e "${GREEN}Cleanup complete${NC}"
}
trap cleanup EXIT

# Build driver
echo -e "${YELLOW}Building driver...${NC}"
cd "$PROJECT_ROOT"
make build
if [ ! -f "build/plugins/elide-task-driver" ]; then
    echo -e "${RED}Driver build failed${NC}"
    exit 1
fi

# Create symlink
echo -e "${YELLOW}Creating plugin symlink...${NC}"
cd build/plugins
ln -sf elide-task-driver elide 2>/dev/null || true
cd "$PROJECT_ROOT"

# Start stubbed server
echo -e "${YELLOW}Starting stubbed server...${NC}"
cd "$PROJECT_ROOT"
make server &
SERVER_PID=$!
sleep 2

# Check if server is running
if [ ! -S /tmp/elide-daemon.sock ]; then
    echo -e "${RED}Server failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}Server started (PID: $SERVER_PID)${NC}"

# Start Nomad
echo -e "${YELLOW}Starting Nomad agent...${NC}"
cd "$PROJECT_ROOT"
nomad agent -dev -config=nomad-agent.hcl &
NOMAD_PID=$!
sleep 5

# Check if driver loaded
echo -e "${YELLOW}Checking if driver loaded...${NC}"
if ! nomad node status -self | grep -q "elide"; then
    echo -e "${RED}Driver did not load${NC}"
    exit 1
fi
echo -e "${GREEN}Driver loaded successfully${NC}"

# Submit test job
echo -e "${YELLOW}Submitting test job...${NC}"
JOB_OUTPUT=$(nomad job run examples/hello-python.nomad 2>&1)
if echo "$JOB_OUTPUT" | grep -q "failed to place"; then
    echo -e "${RED}Job submission failed${NC}"
    echo "$JOB_OUTPUT"
    exit 1
fi

# Extract allocation ID
ALLOC_ID=$(echo "$JOB_OUTPUT" | grep -oP 'Allocation "\K[^"]+' | head -1)
if [ -z "$ALLOC_ID" ]; then
    echo -e "${RED}Failed to get allocation ID${NC}"
    exit 1
fi

echo -e "${GREEN}Job submitted (Allocation: $ALLOC_ID)${NC}"

# Wait for job to complete
echo -e "${YELLOW}Waiting for job to complete...${NC}"
TIMEOUT=30
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS=$(nomad alloc status $ALLOC_ID 2>/dev/null | grep "Client Status" | awk '{print $3}' || echo "unknown")
    if [ "$STATUS" = "complete" ]; then
        echo -e "${GREEN}Job completed successfully${NC}"
        break
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

if [ "$STATUS" != "complete" ]; then
    echo -e "${RED}Job did not complete in time${NC}"
    exit 1
fi

# Check exit code
EXIT_CODE=$(nomad alloc status $ALLOC_ID 2>/dev/null | grep "Exit Code" | awk '{print $3}' || echo "-1")
if [ "$EXIT_CODE" != "0" ]; then
    echo -e "${RED}Job failed with exit code: $EXIT_CODE${NC}"
    exit 1
fi

echo -e "${GREEN}=========================================="
echo -e "All tests passed!${NC}"
echo -e "${GREEN}=========================================="

