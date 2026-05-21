BINARY := git-skill
PKG    := ./cmd/git-skill

.PHONY: build install clean test

build:
	go build -o $(BINARY) $(PKG)

install: build
	cp $(BINARY) $(shell go env GOPATH)/bin/ 2>/dev/null || cp $(BINARY) ~/go/bin/ || cp $(BINARY) /usr/local/bin/

clean:
	rm -f $(BINARY)

test:
	go test ./...
	@bash test.sh
