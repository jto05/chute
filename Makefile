FERDINAND_IMAGE  := chute-ferdinand
SHEET_IMAGE     := chute-sheet

TMUX_DEV_SESSION=chute-dev

# ===================================================================================================================
# Development

run-ferdinand:
	go run ./api/services/ferdinand/...

run-sheet:
	go run ./api/services/sheet/...

# Start both services in a split tmux window (left: ferdinand, right: sheet).
# Kills any existing 'chute' session first so re-running is always clean.
dev:
	tmux kill-session -t ${TMUX_DEV_SESSION} 2>/dev/null || true
	tmux new-session -d -s ${TMUX_DEV_SESSION}
	tmux send-keys -t ${TMUX_DEV_SESSION} 'go run ./api/services/ferdinand/...' Enter
	tmux split-window -h -t ${TMUX_DEV_SESSION}
	tmux send-keys -t ${TMUX_DEV_SESSION} 'go run ./api/services/sheet/...' Enter
	@if [ -n "$$TMUX" ]; then \
		TMUX= tmux attach-session -t ${TMUX_DEV_SESSION}; \
	else \
		tmux attach-session -t ${TMUX_DEV_SESSION}; \
	fi

dev-stop:
	tmux kill-session -t ${TMUX_DEV_SESSION} 2>/dev/null || true

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
