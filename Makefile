# Go params
GOCMD=go
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

all: deps test
test:
	$(GOTEST) -cover -v ./...
clean:
	$(GOCLEAN)
deps:
	$(GOGET) -u

.PHONY: clean all
