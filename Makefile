.DEFAULT_GOAL := test

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
