BINARY := emltomarkdown

.PHONY: all build test run clean fmt vet

all: build

build:
	go build -o $(BINARY) .

test:
	go test ./...

# Pass arguments via ARGS, e.g. `make run ARGS="--input-file mail.eml"`
run:
	go run . $(ARGS)

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	go clean
	rm -f $(BINARY) $(BINARY).exe
