#!/bin/bash

# Docker build script for redis-benchmark-go
# This script builds the Docker image with proper Git information

set -e

# Default values
IMAGE_NAME="redis/redis-benchmark-go"
TAG="latest"
PLATFORM=""
PUSH=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -n, --name NAME       Docker image name (default: redis/redis-benchmark-go)"
    echo "  -t, --tag TAG         Docker image tag (default: latest)"
    echo "  -p, --platform PLATFORM Target platform (e.g., linux/amd64,linux/arm64)"
    echo "  --push                Push image to Docker Hub after building"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Build with defaults (latest tag)"
    echo "  $0 -t v1.0.0 --push                 # Build and push version tag"
    echo "  $0 -p linux/amd64,linux/arm64 --push # Multi-platform build and push"
    echo ""
    echo "Docker Hub Repository: redis/redis-benchmark-go"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Prepare for build
print_info "Preparing Docker build..."

# Build Docker image
FULL_IMAGE_NAME="${IMAGE_NAME}:${TAG}"
print_info "Building Docker image: $FULL_IMAGE_NAME"

# Prepare build command
if [[ -n "$PLATFORM" ]]; then
    # Multi-platform build requires buildx
    print_info "Target platform(s): $PLATFORM"
    print_info "Setting up Docker Buildx for multi-platform build..."

    # Create buildx builder if it doesn't exist
    if ! docker buildx ls | grep -q "multiplatform"; then
        print_info "Creating multiplatform builder..."
        docker buildx create --name multiplatform --use --bootstrap
    else
        print_info "Using existing multiplatform builder..."
        docker buildx use multiplatform
    fi

    BUILD_CMD="docker buildx build --platform $PLATFORM"
    if [[ "$PUSH" == "true" ]]; then
        BUILD_CMD="$BUILD_CMD --push"
    else
        BUILD_CMD="$BUILD_CMD --load"
        print_warning "Multi-platform builds without --push will only load the native platform image"
    fi
else
    # Single platform build uses regular docker build
    BUILD_CMD="docker build"
fi

# Get Git information for build args
GIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(git diff --no-ext-diff 2>/dev/null | wc -l || echo "0")

print_info "Git SHA: $GIT_SHA"
print_info "Git dirty files: $GIT_DIRTY"

# Add build args and tags
BUILD_CMD="$BUILD_CMD --build-arg GIT_SHA=$GIT_SHA --build-arg GIT_DIRTY=$GIT_DIRTY -t $FULL_IMAGE_NAME ."

print_info "Executing: $BUILD_CMD"

# Execute build
if eval $BUILD_CMD; then
    print_info "✅ Docker image built successfully: $FULL_IMAGE_NAME"

    # Show image size (only for single platform builds or when image is loaded locally)
    if [[ -z "$PLATFORM" ]] || [[ "$PUSH" != "true" ]]; then
        IMAGE_SIZE=$(docker images --format "table {{.Size}}" $FULL_IMAGE_NAME 2>/dev/null | tail -n 1)
        if [[ -n "$IMAGE_SIZE" && "$IMAGE_SIZE" != "SIZE" ]]; then
            print_info "Image size: $IMAGE_SIZE"
        fi
    fi

    # Handle push for single platform builds (multi-platform builds push automatically with buildx)
    if [[ "$PUSH" == "true" && -z "$PLATFORM" ]]; then
        print_info "🚀 Pushing image to Docker Hub..."

        # Check if logged in to Docker Hub
        if ! docker info | grep -q "Username:"; then
            print_warning "Not logged in to Docker Hub. Please run: docker login"
            print_info "Or set DOCKER_USERNAME and DOCKER_PASSWORD environment variables"

            if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
                print_info "Using environment variables for Docker login..."
                echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
            else
                print_error "❌ Docker Hub authentication required"
                exit 1
            fi
        fi

        # Push the image
        if docker push $FULL_IMAGE_NAME; then
            print_info "✅ Image pushed successfully to Docker Hub: $FULL_IMAGE_NAME"
        else
            print_error "❌ Failed to push image to Docker Hub"
            exit 1
        fi
    elif [[ "$PUSH" == "true" && -n "$PLATFORM" ]]; then
        print_info "✅ Multi-platform image pushed successfully to Docker Hub: $FULL_IMAGE_NAME"
    fi

    echo ""
    print_info "To run the container:"
    echo "  docker run --rm $FULL_IMAGE_NAME --help"
    echo ""
    print_info "To run with Redis connection:"
    echo "  docker run --rm --network=host $FULL_IMAGE_NAME -h localhost -c 50 -n 100000"
    echo ""
    print_info "To run with custom Redis server:"
    echo "  docker run --rm $FULL_IMAGE_NAME -h redis-server -p 6379 -c 100 -n 1000000"
    echo ""
    if [[ "$PUSH" == "true" ]]; then
        print_info "Image available on Docker Hub: https://hub.docker.com/r/redis/redis-benchmark-go"
    fi
else
    print_error "❌ Docker build failed"
    exit 1
fi
