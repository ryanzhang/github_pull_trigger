# Makefile for github_pull_trigger project

BINARY_NAME=bin/github_pull_trigger
IMAGE_NAME = quay.io/rzhang/github-pull-trigger
IMAGE_TAG = v0.2


.PHONY: all clean deps build image

all: build

#Build the image
image:
	podman build  --no-cache  --platform linux/amd64 -t $(IMAGE_NAME)-amd64:$(IMAGE_TAG) .
	podman build  --no-cache  --platform linux/arm64 -t $(IMAGE_NAME)-arm64:$(IMAGE_TAG) .

push:
	podman tag $(IMAGE_NAME)-amd64:$(IMAGE_TAG) $(IMAGE_NAME):$(IMAGE_TAG)
	podman push $(IMAGE_NAME):$(IMAGE_TAG) --tls-verify=false

test:


# Build the Go binary
build:
	# mkdir -f bin
	GOOS=linux go build -o $(BINARY_NAME)-linux main.go
	GOOS=darwin go build -o $(BINARY_NAME)-darwin main.go
	GOOS=windows go build -o  $(BINARY_NAME)-windows.exe main.go

# Install dependencies
deps:
	go mod tidy

# Clean up build files
clean:
	rm -f $(BINARY_NAME)-linux $(BINARY_NAME)-darwin $(BINARY_NAME)-windows.exe
	rm -rf bin

# Run tests (if applicable)
fulltest:
	go test -v ./...

test:
	go test -v -run TestCrdFile ./...
