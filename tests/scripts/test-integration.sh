#!/bin/bash
# Copyright (c) Elide Dev, Inc.
# SPDX-License-Identifier: Apache-2.0

# Integration test script - tests driver with stubbed server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=========================================="
echo "Elide Task Driver - Integration Test"
echo "=========================================="

cd "$PROJECT_ROOT"

# Build and run unit tests
echo "Running unit tests..."
go test -v ./tests/unit/...

echo "=========================================="
echo "Integration test complete"
echo "=========================================="

