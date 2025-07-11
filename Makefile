BINARY=home-lab-exporter
BINDIR=./bin
SRC=main.go

.PHONY: all build clean install run build-image

all: build

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/$(BINARY) $(SRC)
	
install: build
	install -m 0755 $(BINDIR)/$(BINARY) /usr/local/bin/$(BINARY)

run: build
	$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINDIR)/$(BINARY)

build-image:
	podman manifest create home-lab-exporter || true
	podman build --platform linux/amd64,linux/arm64 --manifest home-lab-exporter:latest .
	podman manifest push --all home-lab-exporter:latest docker://quay.io/cldmnky/home-lab-exporter:latest
