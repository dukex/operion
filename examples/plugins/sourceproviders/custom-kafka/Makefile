clear:
	rm -f plugin.so
.PHONY: clear

./plugin.so: plugin.go
	@echo "Building Custom Kafka source provider plugin..."
	@go build -buildmode=plugin -o plugin.so plugin.go

build: clear ./plugin.so
	@echo "Custom Kafka source provider plugin built successfully."
.PHONY: build