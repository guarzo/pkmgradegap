#!/bin/bash

# Test Environment Setup Script
# Sets up environment variables for testing the Pokemon Grade Gap analyzer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if environment variable is set
check_env_var() {
    local var_name=$1
    local var_value=${!var_name}

    if [ -n "$var_value" ]; then
        print_success "$var_name is set"
        return 0
    else
        print_warning "$var_name is not set, using default"
        return 1
    fi
}

# Function to set default test environment
setup_test_defaults() {
    print_info "Setting up default test environment..."

    # Enable test mode
    export TEST_MODE="true"

    # Set default test tokens if not provided
    export TEST_PRICECHARTING_TOKEN="${TEST_PRICECHARTING_TOKEN:-test-token}"
    export TEST_EBAY_APP_ID="${TEST_EBAY_APP_ID:-test-key}"
    export TEST_POKEMON_API_KEY="${TEST_POKEMON_API_KEY:-test-key}"
    export TEST_PSA_API_KEY="${TEST_PSA_API_KEY:-test-key}"

    print_success "Default test environment configured"
}

# Function to enable mock providers
setup_mock_providers() {
    print_info "Enabling mock providers for testing..."

    export GAMESTOP_MOCK="true"
    export SALES_MOCK="true"
    export POPULATION_MOCK="true"

    print_success "Mock providers enabled"
}

# Function to validate test environment
validate_test_env() {
    print_info "Validating test environment..."

    local all_valid=true

    # Check test mode
    if [ "$TEST_MODE" = "true" ]; then
        print_success "TEST_MODE is enabled"
    else
        print_error "TEST_MODE is not enabled"
        all_valid=false
    fi

    # Check test tokens
    check_env_var "TEST_PRICECHARTING_TOKEN"
    check_env_var "TEST_EBAY_APP_ID"
    check_env_var "TEST_POKEMON_API_KEY"
    check_env_var "TEST_PSA_API_KEY"

    # Check mock providers
    if [ "$GAMESTOP_MOCK" = "true" ]; then
        print_success "GameStop mock enabled"
    fi

    if [ "$SALES_MOCK" = "true" ]; then
        print_success "Sales mock enabled"
    fi

    if [ "$POPULATION_MOCK" = "true" ]; then
        print_success "Population mock enabled"
    fi

    if [ "$all_valid" = true ]; then
        print_success "Test environment validation passed"
        return 0
    else
        print_error "Test environment validation failed"
        return 1
    fi
}

# Function to run tests with current environment
run_tests() {
    local test_type=$1

    print_info "Running tests..."

    case $test_type in
        "unit")
            print_info "Running unit tests..."
            go test -v ./internal/...
            ;;
        "integration")
            print_info "Running integration tests..."
            go test -v ./internal/integration/
            ;;
        "all")
            print_info "Running all tests..."
            go test -v ./...
            ;;
        "coverage")
            print_info "Running tests with coverage..."
            go test -cover ./...
            ;;
        *)
            print_info "Running default test suite..."
            go test ./...
            ;;
    esac
}

# Function to export environment for GitHub Actions
export_github_env() {
    if [ -n "$GITHUB_ENV" ]; then
        print_info "Exporting environment variables for GitHub Actions..."

        echo "TEST_MODE=true" >> $GITHUB_ENV
        echo "GAMESTOP_MOCK=true" >> $GITHUB_ENV
        echo "SALES_MOCK=true" >> $GITHUB_ENV
        echo "POPULATION_MOCK=true" >> $GITHUB_ENV

        # Only export test tokens if they're not default values
        if [ "$TEST_PRICECHARTING_TOKEN" != "test-token" ]; then
            echo "TEST_PRICECHARTING_TOKEN=$TEST_PRICECHARTING_TOKEN" >> $GITHUB_ENV
        fi

        if [ "$TEST_EBAY_APP_ID" != "test-key" ]; then
            echo "TEST_EBAY_APP_ID=$TEST_EBAY_APP_ID" >> $GITHUB_ENV
        fi

        print_success "Environment exported to GitHub Actions"
    fi
}

# Function to create test configuration file
create_test_config() {
    local config_file="${1:-test.env}"

    print_info "Creating test configuration file: $config_file"

    cat > "$config_file" << EOF
# Test Environment Configuration
# Source this file to set up test environment: source $config_file

# Enable test mode
export TEST_MODE="true"

# Test API tokens (replace with real values for integration testing)
export TEST_PRICECHARTING_TOKEN="${TEST_PRICECHARTING_TOKEN:-test-token}"
export TEST_EBAY_APP_ID="${TEST_EBAY_APP_ID:-test-key}"
export TEST_POKEMON_API_KEY="${TEST_POKEMON_API_KEY:-test-key}"
export TEST_PSA_API_KEY="${TEST_PSA_API_KEY:-test-key}"

# Enable mock providers
export GAMESTOP_MOCK="true"
export SALES_MOCK="true"
export POPULATION_MOCK="true"

# Optional: Enable debug logging
# export LOG_LEVEL="debug"
EOF

    print_success "Test configuration file created: $config_file"
    print_info "To use: source $config_file"
}

# Function to clean test environment
clean_test_env() {
    print_info "Cleaning test environment..."

    unset TEST_MODE
    unset TEST_PRICECHARTING_TOKEN
    unset TEST_EBAY_APP_ID
    unset TEST_POKEMON_API_KEY
    unset TEST_PSA_API_KEY
    unset GAMESTOP_MOCK
    unset SALES_MOCK
    unset POPULATION_MOCK

    print_success "Test environment cleaned"
}

# Function to show help
show_help() {
    cat << EOF
Test Environment Setup Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    setup           Set up default test environment
    validate        Validate current test environment
    test [type]     Run tests (unit|integration|all|coverage)
    github          Export environment for GitHub Actions
    config [file]   Create test configuration file
    clean           Clean test environment
    help            Show this help message

Examples:
    $0 setup                    # Set up test environment with defaults
    $0 validate                 # Check current environment
    $0 test unit               # Run unit tests
    $0 test coverage           # Run tests with coverage
    $0 config test.env         # Create test configuration file
    $0 github                  # Export for GitHub Actions

Environment Variables:
    TEST_PRICECHARTING_TOKEN   PriceCharting API test token
    TEST_EBAY_APP_ID          eBay API test app ID
    TEST_POKEMON_API_KEY      Pokemon TCG API test key
    TEST_PSA_API_KEY          PSA API test key
    TEST_MODE                 Enable test mode (true/false)
    GAMESTOP_MOCK            Enable GameStop mock (true/false)
    SALES_MOCK               Enable Sales mock (true/false)
    POPULATION_MOCK          Enable Population mock (true/false)

EOF
}

# Main script logic
main() {
    local command=${1:-setup}

    case $command in
        "setup")
            setup_test_defaults
            setup_mock_providers
            validate_test_env
            ;;
        "validate")
            validate_test_env
            ;;
        "test")
            run_tests "$2"
            ;;
        "github")
            setup_test_defaults
            setup_mock_providers
            export_github_env
            ;;
        "config")
            create_test_config "$2"
            ;;
        "clean")
            clean_test_env
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"