#!/bin/bash

# Start lacylights-fe frontend server on port 3001
# IMPORTANT: NEXT_PUBLIC_* vars are baked at build time, so we must rebuild
# if the port configuration changes.

FRONTEND_DIR="$(dirname "$0")/../../lacylights-fe"
PORT=3001
BACKEND_PORT=4001
CONFIG_FILE=".e2e-build-config"

cd "$FRONTEND_DIR" || { echo "Error: Cannot find lacylights-fe directory"; exit 1; }

export PORT=$PORT
export NEXT_PUBLIC_GRAPHQL_URL="http://localhost:$BACKEND_PORT/graphql"
export NEXT_PUBLIC_GRAPHQL_WS_URL="ws://localhost:$BACKEND_PORT/graphql"

# Check if we need to rebuild
# NEXT_PUBLIC_* vars are baked at build time, so we track what was used
CURRENT_CONFIG="GRAPHQL=$NEXT_PUBLIC_GRAPHQL_URL WS=$NEXT_PUBLIC_GRAPHQL_WS_URL"
NEEDS_BUILD=false

if [ ! -d "out" ]; then
  echo "No static build found, will build..."
  NEEDS_BUILD=true
elif [ ! -f "$CONFIG_FILE" ]; then
  echo "No build config found, will rebuild to ensure correct ports..."
  NEEDS_BUILD=true
elif [ "$(cat "$CONFIG_FILE")" != "$CURRENT_CONFIG" ]; then
  echo "Build config changed, will rebuild..."
  NEEDS_BUILD=true
fi

if [ "$NEEDS_BUILD" = true ]; then
  echo "Building frontend static export with:"
  echo "  NEXT_PUBLIC_GRAPHQL_URL=$NEXT_PUBLIC_GRAPHQL_URL"
  echo "  NEXT_PUBLIC_GRAPHQL_WS_URL=$NEXT_PUBLIC_GRAPHQL_WS_URL"

  # Remove old build and Next.js cache to ensure clean build
  rm -rf out .next

  # Build with correct env vars
  npm run build:static

  # Save config for next time
  echo "$CURRENT_CONFIG" > "$CONFIG_FILE"
fi

echo "Starting lacylights-fe on port $PORT..."
echo "  NEXT_PUBLIC_GRAPHQL_URL=$NEXT_PUBLIC_GRAPHQL_URL"
echo "  NEXT_PUBLIC_GRAPHQL_WS_URL=$NEXT_PUBLIC_GRAPHQL_WS_URL"

npm run serve:static -- -p $PORT
