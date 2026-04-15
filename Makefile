FERDINAND_IMAGE  := chute-ferdinand
SHEET_IMAGE     := chute-sheet

# ===================================================================================================================
# Development

run-ferdinand:
	go run ./api/services/ferdinand/...

run-sheet:
	go run ./api/services/sheet/...

# Start both services in a split tmux window (left: ferdinand, right: sheet).
# Kills any existing 'chute' session first so re-running is always clean.
dev:
	tmux kill-session -t chute 2>/dev/null || true
	tmux new-session -d -s chute
	tmux send-keys -t chute 'go run ./api/services/ferdinand/...' Enter
	tmux split-window -h -t chute
	tmux send-keys -t chute 'go run ./api/services/sheet/...' Enter
	tmux attach-session -t chute

dev-stop:
	tmux kill-session -t chute 2>/dev/null || true

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

# ===================================================================================================================
# Clean

clean:
	rm -rf ./bin
	rm -rf ./data/results
	go clean
