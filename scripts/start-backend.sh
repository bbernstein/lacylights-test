#!/bin/bash

# Start lacylights-go backend server on port 4001

BACKEND_DIR="$(dirname "$0")/../../lacylights-go"
PORT=4001
FRONTEND_PORT=3001

cd "$BACKEND_DIR" || { echo "Error: Cannot find lacylights-go directory"; exit 1; }

export PORT=$PORT
export GRAPHQL_PORT=$PORT
export CORS_ORIGIN="http://localhost:$FRONTEND_PORT"

echo "Starting lacylights-go on port $PORT..."
echo "  CORS_ORIGIN=$CORS_ORIGIN"
make run
