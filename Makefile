# Go params
GOCMD=go
GOCLEAN=$(GOCMD) clean
GOFMT=gofmt
GOGET=$(GOCMD) get
GOTEST=$(GOCMD) test

all: deps fmt test

fmt:
	$(GOFMT) -s -l .

test:
	$(GOTEST) -cover -v ./...

clean:
	$(GOCLEAN)

deps:
	$(GOGET) -u

.PHONY: clean all
