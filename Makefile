test: test-backend

test-%:
	go test -C $(@:test-%=%) -v ./...

.PHONY: helm-lint
helm-lint:
	helm lint ./deploy/helm

.PHONY: helm-unittest
helm-unittest:
	helm plugin install --version=0.8.2 https://github.com/helm-unittest/helm-unittest.git || true
	helm unittest ./deploy/helm
