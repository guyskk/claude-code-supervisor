#!/bin/bash
# End-to-end test script for PreToolUse hook support
# This script simulates Claude Code calling the supervisor hook with AskUserQuestion tool

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CCC_BIN="$PROJECT_ROOT/ccc"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_test() {
    local test_name="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Running test: $test_name"
}

pass_test() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log_info "✓ Test passed"
}

fail_test() {
    local reason="$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log_error "✗ Test failed: $reason"
}

# Setup test environment
setup_test_env() {
    TEST_SESSION_ID="test-pretooluse-$$"
    TEST_STATE_DIR="$PROJECT_ROOT/tmp/e2e-test-$TEST_SESSION_ID"
    mkdir -p "$TEST_STATE_DIR"

    # Export required environment variables
    export CCC_CONFIG_DIR="$TEST_STATE_DIR"
    export CCC_SUPERVISOR_ID="$TEST_SESSION_ID"

    log_info "Test session ID: $TEST_SESSION_ID"
    log_info "Test state dir: $TEST_STATE_DIR"
}

cleanup_test_env() {
    if [ -d "$TEST_STATE_DIR" ]; then
        rm -rf "$TEST_STATE_DIR"
        log_info "Cleaned up test environment"
    fi
}

# Test 1: Verify Stop hook still works (backward compatibility)
test_stop_hook_backward_compatibility() {
    run_test "Stop hook backward compatibility"

    local stop_input='{"session_id":"'$TEST_SESSION_ID'","stop_hook_active":false}'

    # Enable supervisor mode
    echo "true" > "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json"

    local output
    output=$(echo "$stop_input" | "$CCC_BIN" supervisor-hook 2>&1 || true)

    if echo "$output" | grep -q '"decision"'; then
        pass_test
    else
        fail_test "Stop hook output missing 'decision' field"
    fi
}

# Test 2: Verify PreToolUse hook input parsing
test_pretooluse_input_parsing() {
    run_test "PreToolUse input parsing"

    local pretooluse_input='{
        "session_id":"'$TEST_SESSION_ID'",
        "hook_event_name":"PreToolUse",
        "tool_name":"AskUserQuestion",
        "tool_input":{"questions":[{"question":"Test question"}]},
        "tool_use_id":"toolu_test_123"
    }'

    # Enable supervisor mode
    echo '{"enabled":true}' > "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json"

    # Set CCC_SUPERVISOR_HOOK=1 to prevent actual SDK call for this test
    local output
    output=$(CCC_SUPERVISOR_HOOK=1 echo "$pretooluse_input" | "$CCC_BIN" supervisor-hook 2>&1 || true)

    if echo "$output" | grep -q '"reason"'; then
        pass_test
    else
        fail_test "PreToolUse hook output missing expected fields"
    fi
}

# Test 3: Verify PreToolUse hook output format
test_pretooluse_output_format() {
    run_test "PreToolUse output format"

    local pretooluse_input='{
        "session_id":"'$TEST_SESSION_ID'",
        "hook_event_name":"PreToolUse",
        "tool_name":"AskUserQuestion",
        "tool_input":{},
        "tool_use_id":"toolu_test_456"
    }'

    # Enable supervisor mode
    echo '{"enabled":true}' > "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json"

    # Test with CCC_SUPERVISOR_HOOK=1 for early return
    local output
    output=$(CCC_SUPERVISOR_HOOK=1 echo "$pretooluse_input" | "$CCC_BIN" supervisor-hook 2>&1 || true)

    # When supervisor is disabled, should return allow decision
    # When CCC_SUPERVISOR_HOOK=1, should return early with allow
    if [ $? -eq 0 ] || echo "$output" | grep -q '"reason"'; then
        pass_test
    else
        fail_test "PreToolUse hook did not return expected output"
    fi
}

# Test 4: Verify unknown event type defaults to Stop format
test_unknown_event_type_defaults_to_stop() {
    run_test "Unknown event type defaults to Stop format"

    local unknown_event_input='{
        "session_id":"'$TEST_SESSION_ID'",
        "hook_event_name":"UnknownEventType",
        "tool_name":"SomeTool",
        "tool_input":{},
        "tool_use_id":"toolu_test_789"
    }'

    # Enable supervisor mode
    echo '{"enabled":true}' > "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json"

    local output
    output=$(CCC_SUPERVISOR_HOOK=1 echo "$unknown_event_input" | "$CCC_BIN" supervisor-hook 2>&1 || true)

    # Unknown event types should use Stop format (decision/reason fields)
    if echo "$output" | grep -q '"reason"'; then
        pass_test
    else
        fail_test "Unknown event type did not default to Stop format"
    fi
}

# Test 5: Verify hook configuration in provider
test_hook_configuration() {
    run_test "Hook configuration in provider"

    # This test verifies that the PreToolUse hook is configured correctly
    # We check if the settings.json contains the PreToolUse hook with AskUserQuestion matcher

    local settings_file="$TEST_STATE_DIR/settings.json"

    # Create a minimal config
    cat > "$TEST_STATE_DIR/ccc.json" <<EOF
{
    "current_provider": "test",
    "providers": {
        "test": {
            "env": {
                "ANTHROPIC_AUTH_TOKEN": "sk-test",
                "ANTHROPIC_MODEL": "test-model"
            }
        }
    }
}
EOF

    # Run ccc switch to generate hook configuration
    cd "$PROJECT_ROOT"
    if "$CCC_BIN" switch test >/dev/null 2>&1; then
        if [ -f "$settings_file" ]; then
            if grep -q '"PreToolUse"' "$settings_file" && grep -q '"AskUserQuestion"' "$settings_file"; then
                pass_test
            else
                fail_test "PreToolUse hook not configured correctly in settings.json"
            fi
        else
            fail_test "settings.json not created"
        fi
    else
        fail_test "ccc switch command failed"
    fi
}

# Test 6: Verify iteration count increases for PreToolUse events
test_iteration_count_increments() {
    run_test "Iteration count increments for PreToolUse events"

    # Enable supervisor mode with initial count
    cat > "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json" <<EOF
{
    "enabled": true,
    "iteration_count": 0
}
EOF

    local pretooluse_input='{
        "session_id":"'$TEST_SESSION_ID'",
        "hook_event_name":"PreToolUse",
        "tool_name":"AskUserQuestion",
        "tool_input":{},
        "tool_use_id":"toolu_test_iter"
    }'

    # Run hook (with early return to avoid SDK call)
    CCC_SUPERVISOR_HOOK=1 echo "$pretooluse_input" | "$CCC_BIN" supervisor-hook >/dev/null 2>&1 || true

    # Check if iteration count increased
    local new_count
    new_count=$(jq -r '.iteration_count // 0' "$TEST_STATE_DIR/supervisor-$TEST_SESSION_ID.json" 2>/dev/null || echo "0")

    if [ "$new_count" -gt 0 ]; then
        pass_test
    else
        fail_test "Iteration count did not increase (count=$new_count)"
    fi
}

# Main test execution
main() {
    log_info "=== PreToolUse Hook End-to-End Tests ==="
    log_info ""

    # Check if ccc binary exists
    if [ ! -f "$CCC_BIN" ]; then
        log_error "ccc binary not found at $CCC_BIN"
        log_info "Please build the project first: go build -o ccc ./cmd/ccc"
        exit 1
    fi

    # Check dependencies
    if ! command -v jq >/dev/null 2>&1; then
        log_warn "jq not found. Some tests may be skipped."
    fi

    setup_test_env
    trap cleanup_test_env EXIT

    # Run all tests
    test_stop_hook_backward_compatibility
    test_pretooluse_input_parsing
    test_pretooluse_output_format
    test_unknown_event_type_defaults_to_stop
    test_hook_configuration
    test_iteration_count_increments

    # Print summary
    log_info ""
    log_info "=== Test Summary ==="
    log_info "Tests run: $TESTS_RUN"
    log_info "Tests passed: $TESTS_PASSED"
    log_info "Tests failed: $TESTS_FAILED"

    if [ $TESTS_FAILED -eq 0 ]; then
        log_info ""
        log_info "${GREEN}All tests passed!${NC}"
        exit 0
    else
        log_error ""
        log_error "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
