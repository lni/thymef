# Copyright 2023-2024 Lei Ni (nilei81@gmail.com) and other contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all: test-client
PKGNAME=$(shell go list)

.PHONY: test
test:
	go test -v -count=1 .

.PHONY: test-client
test-client:
	go build -o test-client $(PKGNAME)/cmd/client

# static checks
GOLANGCI_LINT_VERSION=v2.1.6
EXTRA_LINTERS=-E misspell -E rowserrcheck -E unconvert -E prealloc
.PHONY: static-check
static-check:
	golangci-lint run --timeout 3m $(EXTRA_LINTERS)

.PHONY: install-static-check-tools
install-static-check-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# clean
.PHONY: clean
clean:
	rm -f test-client
