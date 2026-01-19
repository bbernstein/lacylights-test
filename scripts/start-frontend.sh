#!/bin/bash

# Start lacylights-fe frontend server on port 3001
# Uses Next.js dev server for full functionality during E2E tests

FRONTEND_DIR="$(dirname "$0")/../../lacylights-fe"
PORT=3001
BACKEND_PORT=4001

cd "$FRONTEND_DIR" || { echo "Error: Cannot find lacylights-fe directory"; exit 1; }

export PORT=$PORT
export NEXT_PUBLIC_GRAPHQL_URL="http://localhost:$BACKEND_PORT/graphql"
export NEXT_PUBLIC_GRAPHQL_WS_URL="ws://localhost:$BACKEND_PORT/graphql"

echo "Starting lacylights-fe dev server on port $PORT..."
echo "  NEXT_PUBLIC_GRAPHQL_URL=$NEXT_PUBLIC_GRAPHQL_URL"
echo "  NEXT_PUBLIC_GRAPHQL_WS_URL=$NEXT_PUBLIC_GRAPHQL_WS_URL"

# Use Next.js dev server for full functionality
npm run dev -- -p $PORT
