#!/bin/bash

# Mistigris Advent Calendar launcher for Linux
# This script launches the advent calendar with the correct path to the door32.sys file

# Determine which binary to use based on architecture
ARCH=$(uname -m)
if [ "$ARCH" = "aarch64" ]; then
    BINARY="./advent-linux-arm64"
else
    BINARY="./advent-linux-amd64"
fi

# Use the provided dropfile path
$BINARY -path "$1"

# Exit with the same error code as the advent program
exit $?