
clear:
	rm -f plugin.so
.PHONY: clear

./plugin.so: plugin.go
	@echo "Building plugin..."
	@go build -buildmode=plugin -o plugin.so plugin.go


build: clear ./plugin.so
	@echo "Plugin built successfully."
.PHONY: build