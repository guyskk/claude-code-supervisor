#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default settings
RUN_LINT=false
RUN_TEST=false
RUN_BUILD=false

print_help() {
    echo -e "${BLUE}Usage:${NC} $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -l, --lint          Run lint checks (gofmt, go vet, shellcheck, markdownlint)"
    echo "  -t, --test          Run tests with race detector"
    echo "  -b, --build         Run build validation"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "If no options specified, runs all checks (lint, test, build)."
    echo ""
    echo "Examples:"
    echo "  $0                          # Run all checks"
    echo "  $0 --lint                   # Run lint only"
}

# Check functions
check_gofmt() {
    echo -e "${YELLOW}[1/4]${NC} ${BLUE}Running gofmt...${NC}"

    local unformatted
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo -e "${RED}  ✗ Fail: The following files are not properly formatted:${NC}"
        echo "$unformatted" | while read -r file; do
            echo -e "    ${RED}-${NC} $file"
        done
        echo ""
        echo -e "${YELLOW}Hint: Run 'gofmt -w .' to fix formatting${NC}"
        return 1
    fi

    echo -e "${GREEN}  ✓${NC} All files are properly formatted"
    return 0
}

check_go_vet() {
    echo -e "${YELLOW}[2/4]${NC} ${BLUE}Running go vet...${NC}"

    if go vet ./... 2>&1; then
        echo -e "${GREEN}  ✓${NC} go vet passed"
        return 0
    else
        return 1
    fi
}

check_shellcheck() {
    echo -e "${YELLOW}[3/4]${NC} ${BLUE}Running shellcheck...${NC}"

    if ! command -v shellcheck &>/dev/null; then
        echo -e "${YELLOW}  ⚠${NC} shellcheck not found, skipping shell checks"
        return 0
    fi

    local sh_files
    sh_files=$(find . -maxdepth 1 -name "*.sh" -type f)
    if [ -z "$sh_files" ]; then
        echo -e "${GREEN}  ✓${NC} No shell scripts to check"
        return 0
    fi

    # shellcheck disable=SC2086
    if shellcheck $sh_files 2>&1; then
        echo -e "${GREEN}  ✓${NC} shellcheck passed"
        return 0
    else
        return 1
    fi
}

check_markdown() {
    echo -e "${YELLOW}[4/4]${NC} ${BLUE}Running markdownlint...${NC}"

    if ! command -v markdownlint &>/dev/null; then
        echo -e "${YELLOW}  ⚠${NC} markdownlint not found, skipping markdown checks"
        return 0
    fi

    local md_files
    md_files=$(find . -maxdepth 1 -name "*.md" -type f)
    if [ -z "$md_files" ]; then
        echo -e "${GREEN}  ✓${NC} No markdown files to check"
        return 0
    fi

    # shellcheck disable=SC2086
    if markdownlint $md_files 2>&1; then
        echo -e "${GREEN}  ✓${NC} markdownlint passed"
        return 0
    else
        return 1
    fi
}

run_lint() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                        LINT CHECKS                           ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    check_gofmt || exit 1
    check_go_vet || exit 1
    check_shellcheck || exit 1
    check_markdown || exit 1

    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}                        LINT PASSED                           ${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
}

run_test() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                        TEST CHECKS                           ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    echo -e "${YELLOW}Running: go test -v -race -tags=ci ./...${NC}"

    if go test -v -race -tags=ci ./...; then
        echo ""
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                        TESTS PASSED                         ${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${RED}                        TESTS FAILED                         ${NC}"
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        return 1
    fi
}

run_build() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                       BUILD CHECKS                           ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    if ./build.sh --all; then
        echo ""
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                        BUILD PASSED                         ${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${RED}                        BUILD FAILED                         ${NC}"
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        return 1
    fi
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -l|--lint)
            RUN_LINT=true
            shift
            ;;
        -t|--test)
            RUN_TEST=true
            shift
            ;;
        -b|--build)
            RUN_BUILD=true
            shift
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            print_help
            exit 1
            ;;
    esac
done

# If no specific check selected, run all
if [ "$RUN_LINT" = false ] && [ "$RUN_TEST" = false ] && [ "$RUN_BUILD" = false ]; then
    RUN_LINT=true
    RUN_TEST=true
    RUN_BUILD=true
fi

# Main execution
echo -e "${BLUE}Starting pre-commit checks...${NC}"

EXIT_CODE=0

if [ "$RUN_LINT" = true ]; then
    run_lint || EXIT_CODE=1
fi

if [ "$RUN_TEST" = true ]; then
    run_test || EXIT_CODE=1
fi

if [ "$RUN_BUILD" = true ]; then
    run_build || EXIT_CODE=1
fi

echo ""

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some checks failed${NC}"
    exit 1
fi
