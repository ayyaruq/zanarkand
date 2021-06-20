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
	$(GOTEST) -cover -v $$($(GOCMD) list ./... | grep -v examples)

clean:
	$(GOCLEAN)

deps:
	$(GOGET) -u

.PHONY: clean all
