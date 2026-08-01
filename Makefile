BINARY := emltomarkdown

.PHONY: all build build-win-x64 test run clean fmt vet

all: build

build:
	go build -o $(BINARY) .

build-win-x64:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY).exe .

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
