# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=gcs-info-catch
PID:=$(shell cat ./gcsInfoCatch.pid)
LOG_FILE=./log/gcsInfoCatch.log

all: build
build:
	$(GOBUILD) -o $(BINARY_NAME) *.go
clean:
	$(GOCLEAN)
	rm -rf $(BINARY_NAME)
	#rm -rf $(LOG_FILE)
	kill -9 $(PID)
run:
	$(GOBUILD) -o $(BINARY_NAME) *.go
	./$(BINARY_NAME)
update:
	python scripts/updateset.py
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME) -mode update