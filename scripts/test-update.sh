#!/bin/bash
# Test script for the update command

# Build the binary with version information
echo "Building Netronome with version information..."
go build -ldflags "-X main.Version=v0.1.0 -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o ./netronome-test ./cmd/netronome

echo ""
echo "Testing version command:"
./netronome-test version

echo ""
echo "Testing update command (dry run - won't actually update):"
./netronome-test update

# Clean up
rm -f ./netronome-test