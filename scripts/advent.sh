#!/bin/bash
# BBS Door Launcher Script Template for Linux
# 
# TEMPLATE FILE: Copy this to your BBS door directory and customize paths
# Called by BBS: advent.sh [node] [socket_handle]
#
# Replace paths below with your actual BBS directories

NODE=${1:-1}
SOCKET_HANDLE=${2:-0}

# Change to door directory
cd /opt/bbs/doors/advent

# BBS door32.sys location - adjust path for your BBS
DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"

# Log the launch
echo "[$(date)] Node ${NODE} - Starting Advent Calendar Door" >> advent_door.log

# Check if door32.sys exists
if [ ! -f "$DROPFILE_PATH" ]; then
    echo "[$(date)] ERROR: door32.sys not found at $DROPFILE_PATH" >> advent_door.log
    echo "ERROR: door32.sys file not found."
    echo "Expected location: $DROPFILE_PATH"
    echo "Check your BBS configuration."
    exit 1
fi

# Launch the door
echo "[$(date)] Using door32.sys: $DROPFILE_PATH" >> advent_door.log
./advent --path "$DROPFILE_PATH"

# Log completion
echo "[$(date)] Node ${NODE} - Door session ended" >> advent_door.log