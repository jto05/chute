FERDINAND_IMAGE  := chute-ferdinand
SHEET_IMAGE     := chute-sheet

# ===================================================================================================================
# Development

run-ferdinand:
	go run ./api/services/ferdinand/...

run-sheet:
	go run ./api/services/sheet/...

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./... -count=1

# ===================================================================================================================
# Build

build-ferdinand:
	go build -o ./bin/ferdinand ./api/services/ferdinand/...

build-sheet:
	go build -o ./bin/sheet ./api/services/sheet/...

# ===================================================================================================================
# Docker

docker-ferdinand:
	docker build -f zarf/docker/Dockerfile.ferdinand -t $(FERDINAND_IMAGE) .

docker-sheet:
	docker build -f zarf/docker/Dockerfile.sheet -t $(SHEET_IMAGE) .
