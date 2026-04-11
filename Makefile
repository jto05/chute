SCRAPER_IMAGE  := chute-scraper
SHEET_IMAGE     := chute-sheet

# ===================================================================================================================
# Development

run-scraper:
	go run ./api/services/scraper/...

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

build-scraper:
	go build -o ./bin/scraper ./api/services/scraper/...

build-sheet:
	go build -o ./bin/sheet ./api/services/sheet/...

# ===================================================================================================================
# Docker

docker-scraper:
	docker build -f zarf/docker/Dockerfile.scraper -t $(SCRAPER_IMAGE) .

docker-sheet:
	docker build -f zarf/docker/Dockerfile.sheet -t $(SHEET_IMAGE) .
