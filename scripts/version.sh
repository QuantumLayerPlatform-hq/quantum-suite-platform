#!/bin/bash

# QLens Semantic Versioning Script
# Manages versions across all artifacts consistently

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="${PROJECT_ROOT}/VERSION"
CHART_FILE="${PROJECT_ROOT}/charts/qlens/Chart.yaml"
COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

echo_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get current version
get_current_version() {
    if [[ -f "${VERSION_FILE}" ]]; then
        cat "${VERSION_FILE}"
    else
        echo "0.0.0"
    fi
}

# Validate semantic version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
        echo_error "Invalid semantic version format: $version"
        echo_info "Expected format: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]"
        exit 1
    fi
}

# Parse semantic version
parse_version() {
    local version=$1
    # Remove pre-release and build metadata for parsing
    local clean_version=${version%%-*}
    clean_version=${clean_version%%+*}
    
    IFS='.' read -ra VERSION_PARTS <<< "$clean_version"
    MAJOR=${VERSION_PARTS[0]}
    MINOR=${VERSION_PARTS[1]}
    PATCH=${VERSION_PARTS[2]}
    
    # Extract pre-release and build metadata
    PRE_RELEASE=""
    BUILD_METADATA=""
    
    if [[ $version == *"-"* ]]; then
        PRE_RELEASE=${version#*-}
        PRE_RELEASE=${PRE_RELEASE%+*}
    fi
    
    if [[ $version == *"+"* ]]; then
        BUILD_METADATA=${version#*+}
    fi
}

# Increment version
increment_version() {
    local current_version=$1
    local increment_type=$2
    
    parse_version "$current_version"
    
    case $increment_type in
        "major")
            MAJOR=$((MAJOR + 1))
            MINOR=0
            PATCH=0
            ;;
        "minor")
            MINOR=$((MINOR + 1))
            PATCH=0
            ;;
        "patch")
            PATCH=$((PATCH + 1))
            ;;
        *)
            echo_error "Invalid increment type: $increment_type"
            echo_info "Valid types: major, minor, patch"
            exit 1
            ;;
    esac
    
    echo "${MAJOR}.${MINOR}.${PATCH}"
}

# Update VERSION file
update_version_file() {
    local version=$1
    echo "$version" > "$VERSION_FILE"
    echo_success "Updated VERSION file to $version"
}

# Update Helm Chart version
update_helm_chart() {
    local version=$1
    local app_version=$1
    
    if [[ -f "$CHART_FILE" ]]; then
        # Update chart version
        sed -i.bak "s/^version:.*/version: $version/" "$CHART_FILE"
        # Update app version
        sed -i.bak "s/^appVersion:.*/appVersion: \"$app_version\"/" "$CHART_FILE"
        rm -f "${CHART_FILE}.bak"
        echo_success "Updated Helm chart to version $version (appVersion: $app_version)"
    else
        echo_warning "Helm chart file not found: $CHART_FILE"
    fi
}

# Update Docker Compose
update_docker_compose() {
    local version=$1
    
    if [[ -f "$COMPOSE_FILE" ]]; then
        # Update image tags in docker-compose.yml
        sed -i.bak "s|image: ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-.*:.*|image: ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-gateway:$version|g" "$COMPOSE_FILE"
        rm -f "${COMPOSE_FILE}.bak"
        echo_success "Updated Docker Compose to version $version"
    else
        echo_warning "Docker Compose file not found: $COMPOSE_FILE"
    fi
}

# Update Go module version in go.mod
update_go_mod() {
    local version=$1
    
    if [[ -f "${PROJECT_ROOT}/go.mod" ]]; then
        # Add version comment to go.mod
        if ! grep -q "// Version:" "${PROJECT_ROOT}/go.mod"; then
            echo "// Version: $version" >> "${PROJECT_ROOT}/go.mod"
        else
            sed -i.bak "s|// Version:.*|// Version: $version|" "${PROJECT_ROOT}/go.mod"
            rm -f "${PROJECT_ROOT}/go.mod.bak"
        fi
        echo_success "Updated go.mod version comment to $version"
    fi
}

# Create version manifest
create_version_manifest() {
    local version=$1
    local manifest_file="${PROJECT_ROOT}/VERSION_MANIFEST.json"
    
    cat > "$manifest_file" << EOF
{
  "version": "$version",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "git_branch": "$(git branch --show-current 2>/dev/null || echo 'unknown')",
  "artifacts": {
    "helm_chart": {
      "version": "$version",
      "app_version": "$version"
    },
    "docker_images": [
      "ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-gateway:$version",
      "ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-router:$version",
      "ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-cache:$version"
    ],
    "components": {
      "gateway": "$version",
      "router": "$version",
      "cache": "$version",
      "providers": {
        "azure_openai": "$version",
        "aws_bedrock": "$version"
      }
    }
  },
  "compatibility": {
    "kubernetes": ">=1.24.0",
    "helm": ">=3.8.0",
    "go": ">=1.21.0"
  },
  "checksums": {}
}
EOF

    echo_success "Created version manifest: $manifest_file"
}

# Generate build metadata
generate_build_metadata() {
    local base_version=$1
    local git_commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
    local build_time=$(date -u +"%Y%m%d%H%M%S")
    
    echo "${base_version}+${build_time}.${git_commit}"
}

# Tag Git repository
tag_git_repo() {
    local version=$1
    local tag="v$version"
    
    if git rev-parse --git-dir > /dev/null 2>&1; then
        echo_info "Creating Git tag: $tag"
        git tag -a "$tag" -m "Release $version"
        echo_success "Created Git tag: $tag"
        echo_info "Don't forget to push the tag: git push origin $tag"
    else
        echo_warning "Not a Git repository, skipping tag creation"
    fi
}

# Show current version information
show_version_info() {
    local current_version=$(get_current_version)
    
    echo_info "Current QLens Version Information"
    echo "=================================="
    echo "Version: $current_version"
    
    if [[ -f "${PROJECT_ROOT}/VERSION_MANIFEST.json" ]]; then
        echo ""
        echo "Manifest Information:"
        jq -r '.timestamp' "${PROJECT_ROOT}/VERSION_MANIFEST.json" | xargs -I {} echo "Timestamp: {}"
        jq -r '.git_commit' "${PROJECT_ROOT}/VERSION_MANIFEST.json" | xargs -I {} echo "Git Commit: {}"
        jq -r '.git_branch' "${PROJECT_ROOT}/VERSION_MANIFEST.json" | xargs -I {} echo "Git Branch: {}"
        echo ""
        echo "Artifacts:"
        jq -r '.artifacts.docker_images[]' "${PROJECT_ROOT}/VERSION_MANIFEST.json" | xargs -I {} echo "  - {}"
    fi
    
    echo ""
    echo "Files managed:"
    [[ -f "$VERSION_FILE" ]] && echo "  ✓ VERSION"
    [[ -f "$CHART_FILE" ]] && echo "  ✓ Helm Chart (charts/qlens/Chart.yaml)"
    [[ -f "$COMPOSE_FILE" ]] && echo "  ✓ Docker Compose"
    [[ -f "${PROJECT_ROOT}/go.mod" ]] && echo "  ✓ Go Module"
    [[ -f "${PROJECT_ROOT}/VERSION_MANIFEST.json" ]] && echo "  ✓ Version Manifest"
}

# Main release function
release_version() {
    local increment_type=$1
    local current_version=$(get_current_version)
    local new_version
    
    if [[ "$increment_type" == "current" ]]; then
        new_version=$current_version
    else
        new_version=$(increment_version "$current_version" "$increment_type")
    fi
    
    validate_version "$new_version"
    
    echo_info "Preparing release: $current_version → $new_version"
    
    # Update all version files
    update_version_file "$new_version"
    update_helm_chart "$new_version"
    update_docker_compose "$new_version"
    update_go_mod "$new_version"
    create_version_manifest "$new_version"
    
    echo_success "Successfully updated all artifacts to version $new_version"
    
    # Optionally create Git tag
    read -p "Create Git tag v$new_version? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        tag_git_repo "$new_version"
    fi
    
    echo ""
    echo_info "Release Summary:"
    echo "  Version: $new_version"
    echo "  Docker Images: ghcr.io/quantumlayerplatform/quantumlayerplatform/qlens-*:$new_version"
    echo "  Helm Chart: charts/qlens:$new_version"
    echo ""
    echo_info "Next Steps:"
    echo "  1. Review changes: git diff"
    echo "  2. Commit changes: git add . && git commit -m 'chore: release v$new_version'"
    echo "  3. Push changes: git push origin main"
    echo "  4. Push tag: git push origin v$new_version"
    echo "  5. Create GitHub release with changelog"
}

# Set pre-release version
set_prerelease() {
    local prerelease_type=$1
    local current_version=$(get_current_version)
    
    # Remove any existing pre-release suffix
    local base_version=${current_version%%-*}
    base_version=${base_version%%+*}
    
    local new_version="${base_version}-${prerelease_type}"
    
    validate_version "$new_version"
    
    echo_info "Setting pre-release version: $current_version → $new_version"
    
    update_version_file "$new_version"
    update_helm_chart "$new_version"
    create_version_manifest "$new_version"
    
    echo_success "Set pre-release version: $new_version"
}

# Usage information
usage() {
    echo "QLens Version Management Script"
    echo ""
    echo "Usage: $0 COMMAND [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  info                    Show current version information"
    echo "  release LEVEL          Release new version (major|minor|patch)"
    echo "  prerelease TYPE        Set pre-release version (alpha|beta|rc.N)"
    echo "  set VERSION            Set specific version"
    echo "  build-metadata         Generate version with build metadata"
    echo "  validate VERSION       Validate semantic version format"
    echo ""
    echo "Examples:"
    echo "  $0 info                           # Show current version"
    echo "  $0 release patch                  # Release patch version (1.0.0 → 1.0.1)"
    echo "  $0 release minor                  # Release minor version (1.0.1 → 1.1.0)"
    echo "  $0 release major                  # Release major version (1.1.0 → 2.0.0)"
    echo "  $0 prerelease alpha               # Set alpha version (1.0.0 → 1.0.0-alpha)"
    echo "  $0 prerelease beta                # Set beta version (1.0.0 → 1.0.0-beta)"
    echo "  $0 prerelease rc.1                # Set release candidate (1.0.0 → 1.0.0-rc.1)"
    echo "  $0 set 2.1.0                     # Set specific version"
    echo "  $0 build-metadata                 # Generate build version"
    echo "  $0 validate 1.2.3-alpha.1+build  # Validate version format"
}

# Main script logic
main() {
    case ${1:-} in
        "info")
            show_version_info
            ;;
        "release")
            if [[ -z ${2:-} ]]; then
                echo_error "Release level required (major|minor|patch)"
                usage
                exit 1
            fi
            release_version "$2"
            ;;
        "prerelease")
            if [[ -z ${2:-} ]]; then
                echo_error "Pre-release type required (alpha|beta|rc.N)"
                usage
                exit 1
            fi
            set_prerelease "$2"
            ;;
        "set")
            if [[ -z ${2:-} ]]; then
                echo_error "Version required"
                usage
                exit 1
            fi
            validate_version "$2"
            update_version_file "$2"
            update_helm_chart "$2"
            update_docker_compose "$2"
            update_go_mod "$2"
            create_version_manifest "$2"
            echo_success "Set version to $2"
            ;;
        "build-metadata")
            current_version=$(get_current_version)
            build_version=$(generate_build_metadata "$current_version")
            echo "$build_version"
            ;;
        "validate")
            if [[ -z ${2:-} ]]; then
                echo_error "Version required"
                usage
                exit 1
            fi
            validate_version "$2"
            echo_success "Version $2 is valid"
            ;;
        "help"|"-h"|"--help"|"")
            usage
            ;;
        *)
            echo_error "Unknown command: $1"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"