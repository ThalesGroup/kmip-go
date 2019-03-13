# Expands to list this project's go packages, excluding the vendor folder
SHELL = bash
PACKAGES = $$(go list ./... | grep -v /vendor/)
BUILD_FLAGS =
TEST_FLAGS = -vet=-all

all: generate fmt build test

build:
	go build $(BUILD_FLAGS) $(PACKAGES)

builddir:
	@if [ ! -d build ]; then mkdir build; fi

vet:
	go vet $(PACKAGES)

lint:
	golint -set_exit_status $(PACKAGES)

clean:
	rm -rf build/*

fmt:
	go fmt $(PACKAGES)

generate:
	go generate $(PACKAGES)

test:
	go test $(BUILD_FLAGS) $(TEST_FLAGS) $(PACKAGES)

testreport: builddir
	# runs go test in each package one at a time, generating coverage profiling
    # finally generates a combined junit test report and a test coverage report
    # note: running coverage messes up line numbers in error stacktraces
	go test $(BUILD_FLAGS) $(TEST_FLAGS) -v -covermode=count -coverprofile=build/coverage.out $(PACKAGES) | tee build/test.out
	go tool cover -html=build/coverage.out -o build/coverage.html
	go2xunit -input build/test.out -output build/test.xml
	@! grep -e "--- FAIL" -e "^FAIL" build/test.out

docker:
	docker-compose build --pull builder
	docker-compose run --rm builder make all testreport

vendor.update:
	dep ensure --update

vendor.ensure:
	dep ensure

### TOOLS

tools: buildtools
	go get -u github.com/golang/dep/cmd/dep

buildtools:
# installs tools used during build
	go get -u github.com/tebeka/go2xunit
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/golang/lint/golint

.PHONY: all build builddir run artifacts vet lint clean fmt test testall testreport up down pull builder runc ci bash fish image prep vendor.update vendor.ensure tools buildtools migratetool db.migrate

