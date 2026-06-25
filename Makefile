.DEFAULT_GOAL := test

run:
	@echo "Starting Aira..."
	@go run main.go

build:
	@echo "Building..."
	@go build -o airo main.go

install-voice-linux:
	@echo "Installing voice dependencies (Linux)..."
	@apt-get install -y libopus-dev libopusfile-dev pkg-config ffmpeg

install-voice-mac:
	@echo "Installing voice dependencies (macOS)..."
	@brew install opus opusfile pkg-config ffmpeg

download-voice-model:
	@echo "Downloading Piper voice model..."
	@curl -LO https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx
	@curl -LO https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
	@echo "Model downloaded. Set PIPER_MODEL to: $(PWD)/en_US-lessac-medium.onnx"

clean-macos-metadata:
	@echo "Removing macOS metadata..."
	@find . -name '.DS_Store' -delete 2>/dev/null || true
	@find . -name '._*' -delete 2>/dev/null || true

cleanall: clean-macos-metadata
	@echo "Full cleanup complete!"

test:
	@echo "Running tests..."
	@go test ./tests/$(if $(FILE),$(FILE),...) -v

tidy:
	@echo "Tidying modules..."
	@go mod tidy
