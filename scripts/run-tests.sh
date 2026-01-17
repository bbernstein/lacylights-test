#!/usr/bin/env bash

# LacyLights Test Runner
# Ensures backend (and frontend for e2e) is running and executes the selected test suite

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/.." || exit 1

# Port configuration
BACKEND_PORT=4001
FRONTEND_PORT=3001
GRAPHQL_ENDPOINT="http://localhost:$BACKEND_PORT/graphql"

# Track what we started
STARTED_BACKEND=false
STARTED_FRONTEND=false
BACKEND_PID=""
FRONTEND_PID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test options (parallel arrays instead of associative arrays for compatibility)
TEST_KEYS=(all ci contracts dmx fade effects preview settings integration distribution load e2e e2e-ui e2e-headed)
TEST_DESCRIPTIONS=(
  "Run all contract tests (requires Art-Net)"
  "Run CI-safe tests (no Art-Net required)"
  "Run API contract tests"
  "Run DMX behavior tests"
  "Run fade behavior tests"
  "Run FX Engine effects tests"
  "Run preview mode tests"
  "Run settings contract tests"
  "Run integration tests"
  "Run S3 distribution tests"
  "Run 4-universe load tests"
  "Run E2E tests (Playwright)"
  "Run E2E tests with Playwright UI"
  "Run E2E tests in headed browser"
)
MAKE_TARGETS=(test test-ci test-contracts test-dmx test-fade test-effects test-preview test-settings test-integration test-distribution test-load e2e e2e-ui e2e-headed)

# Tests that require frontend
FRONTEND_REQUIRED_TESTS="e2e e2e-ui e2e-headed"

get_test_index() {
  local key="$1"
  for i in "${!TEST_KEYS[@]}"; do
    if [[ "${TEST_KEYS[$i]}" == "$key" ]]; then
      echo "$i"
      return 0
    fi
  done
  echo "-1"
  return 1
}

requires_frontend() {
  local test="$1"
  [[ " $FRONTEND_REQUIRED_TESTS " == *" $test "* ]]
}

show_usage() {
  echo -e "${BLUE}LacyLights Test Runner${NC}"
  echo ""
  echo "Usage: $0 [test-type]"
  echo ""
  echo "Available test types:"
  echo ""
  echo -e "${CYAN}Contract & Integration Tests:${NC}"
  for i in 0 1 2 3 4 5 6 7 8 9 10; do
    printf "  ${GREEN}%-15s${NC} %s\n" "${TEST_KEYS[$i]}" "${TEST_DESCRIPTIONS[$i]}"
  done
  echo ""
  echo -e "${CYAN}E2E Tests (require frontend):${NC}"
  for i in 11 12 13; do
    printf "  ${GREEN}%-15s${NC} %s\n" "${TEST_KEYS[$i]}" "${TEST_DESCRIPTIONS[$i]}"
  done
  echo ""
  echo "Examples:"
  echo "  $0              # Show this menu and prompt for selection"
  echo "  $0 all          # Run all contract tests"
  echo "  $0 contracts    # Run contract tests only"
  echo "  $0 ci           # Run CI-safe tests"
  echo "  $0 e2e          # Run E2E Playwright tests"
}

select_test() {
  echo -e "${BLUE}LacyLights Test Runner${NC}"
  echo ""
  echo "Select a test suite to run:"
  echo ""

  echo -e "${CYAN}Contract & Integration Tests:${NC}"
  for i in 0 1 2 3 4 5 6 7 8 9 10; do
    printf "  ${GREEN}%2d)${NC} %-15s %s\n" "$((i+1))" "${TEST_KEYS[$i]}" "${TEST_DESCRIPTIONS[$i]}"
  done
  echo ""
  echo -e "${CYAN}E2E Tests (require frontend):${NC}"
  for i in 11 12 13; do
    printf "  ${GREEN}%2d)${NC} %-15s %s\n" "$((i+1))" "${TEST_KEYS[$i]}" "${TEST_DESCRIPTIONS[$i]}"
  done
  echo ""
  printf "  ${YELLOW} 0)${NC} Exit\n"
  echo ""

  read -rp "Enter selection [0-${#TEST_KEYS[@]}]: " selection

  if [[ "$selection" == "0" ]]; then
    echo "Exiting."
    exit 0
  fi

  if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "${#TEST_KEYS[@]}" ]; then
    SELECTED_INDEX=$((selection-1))
    SELECTED_TEST="${TEST_KEYS[$SELECTED_INDEX]}"
  else
    echo -e "${RED}Invalid selection.${NC}"
    exit 1
  fi
}

# =============================================================================
# Backend Management
# =============================================================================

is_backend_running() {
  curl -sf "$GRAPHQL_ENDPOINT" -X POST \
      -H "Content-Type: application/json" \
      -d '{"query":"{ __typename }"}' > /dev/null 2>&1
}

kill_backend() {
  echo -e "${YELLOW}Stopping backend on port $BACKEND_PORT...${NC}"
  lsof -ti:$BACKEND_PORT | xargs kill -9 2>/dev/null || true
  sleep 1
}

start_backend() {
  echo -e "${YELLOW}Starting backend on port $BACKEND_PORT...${NC}"
  "$SCRIPT_DIR/start-backend.sh" &
  BACKEND_PID=$!

  # Wait for backend to be ready (up to 30 seconds)
  echo "Waiting for backend to start..."
  for i in {1..30}; do
    if is_backend_running; then
      echo -e "${GREEN}Backend is ready.${NC}"
      STARTED_BACKEND=true
      return 0
    fi
    if [ $i -eq 30 ]; then
      echo -e "${RED}Error: Backend failed to start within 30 seconds.${NC}"
      kill $BACKEND_PID 2>/dev/null
      exit 1
    fi
    sleep 1
  done
}

check_and_manage_backend() {
  echo -e "${BLUE}Checking if backend is running on port $BACKEND_PORT...${NC}"

  if is_backend_running; then
    echo -e "${GREEN}Backend is already running on port $BACKEND_PORT.${NC}"
    echo ""
    read -rp "Do you want to restart it? [y/N]: " restart_choice

    if [[ "$restart_choice" =~ ^[Yy]$ ]]; then
      kill_backend
      start_backend
    else
      echo -e "${GREEN}Using existing backend.${NC}"
      STARTED_BACKEND=false
    fi
  else
    echo -e "${YELLOW}Backend not running.${NC}"
    start_backend
  fi
}

# =============================================================================
# Frontend Management
# =============================================================================

is_frontend_running() {
  curl -sf "http://localhost:$FRONTEND_PORT" > /dev/null 2>&1
}

kill_frontend() {
  echo -e "${YELLOW}Stopping frontend on port $FRONTEND_PORT...${NC}"
  lsof -ti:$FRONTEND_PORT | xargs kill -9 2>/dev/null || true
  sleep 1
}

start_frontend() {
  echo -e "${YELLOW}Starting frontend on port $FRONTEND_PORT...${NC}"
  "$SCRIPT_DIR/start-frontend.sh" &
  FRONTEND_PID=$!

  # Wait for frontend to be ready (up to 60 seconds for build)
  echo "Waiting for frontend to start (may take a while if building)..."
  for i in {1..60}; do
    if is_frontend_running; then
      echo -e "${GREEN}Frontend is ready.${NC}"
      STARTED_FRONTEND=true
      return 0
    fi
    if [ $i -eq 60 ]; then
      echo -e "${RED}Error: Frontend failed to start within 60 seconds.${NC}"
      kill $FRONTEND_PID 2>/dev/null
      exit 1
    fi
    sleep 1
  done
}

check_and_manage_frontend() {
  echo -e "${BLUE}Checking if frontend is running on port $FRONTEND_PORT...${NC}"

  if is_frontend_running; then
    echo -e "${GREEN}Frontend is already running on port $FRONTEND_PORT.${NC}"
    echo ""
    read -rp "Do you want to restart it? [y/N]: " restart_choice

    if [[ "$restart_choice" =~ ^[Yy]$ ]]; then
      kill_frontend
      start_frontend
    else
      echo -e "${GREEN}Using existing frontend.${NC}"
      STARTED_FRONTEND=false
    fi
  else
    echo -e "${YELLOW}Frontend not running.${NC}"
    start_frontend
  fi
}

# =============================================================================
# Test Execution
# =============================================================================

run_tests() {
  local index
  index=$(get_test_index "$SELECTED_TEST")
  local target="${MAKE_TARGETS[$index]}"

  echo ""
  echo -e "${BLUE}Running: make $target${NC}"
  echo ""

  GRAPHQL_ENDPOINT=$GRAPHQL_ENDPOINT \
    ARTNET_LISTEN_PORT=6454 \
    ARTNET_BROADCAST=127.0.0.1 \
    make "$target"

  TEST_EXIT_CODE=$?
}

# =============================================================================
# Cleanup
# =============================================================================

cleanup() {
  if [ "$STARTED_FRONTEND" = true ]; then
    echo ""
    echo -e "${YELLOW}Cleaning up: stopping frontend that was started by this script...${NC}"
    kill $FRONTEND_PID 2>/dev/null
    lsof -ti:$FRONTEND_PORT | xargs kill -9 2>/dev/null || true
    echo -e "${GREEN}Frontend stopped.${NC}"
  fi

  if [ "$STARTED_BACKEND" = true ]; then
    echo ""
    echo -e "${YELLOW}Cleaning up: stopping backend that was started by this script...${NC}"
    kill $BACKEND_PID 2>/dev/null
    lsof -ti:$BACKEND_PORT | xargs kill -9 2>/dev/null || true
    echo -e "${GREEN}Backend stopped.${NC}"
  fi
}

# =============================================================================
# Main Script
# =============================================================================

trap cleanup EXIT

# Parse command line argument or show menu
if [ $# -eq 0 ]; then
  select_test
elif [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
  show_usage
  exit 0
else
  index=$(get_test_index "$1")
  if [ "$index" -ge 0 ]; then
    SELECTED_TEST="$1"
  else
    echo -e "${RED}Unknown test type: $1${NC}"
    echo ""
    show_usage
    exit 1
  fi
fi

echo ""
index=$(get_test_index "$SELECTED_TEST")
echo -e "${BLUE}Selected: ${GREEN}$SELECTED_TEST${NC} - ${TEST_DESCRIPTIONS[$index]}"
echo ""

# Check backend status (always needed)
check_and_manage_backend

# Check frontend status (only for e2e tests)
if requires_frontend "$SELECTED_TEST"; then
  echo ""
  check_and_manage_frontend
fi

run_tests

exit $TEST_EXIT_CODE
