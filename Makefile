export GLOG = warn
export BINLOG = warn
export HTTPLOG = warn

test: test_pki test_username test_bully test_avatar

test_pki:
	go test -timeout 5m -v -race -run Test_three_nodes ./peer/tests/integration

test_username:
	go test -v -race -run Test_Username ./peer/tests/unit

test_bully:
	go test -v -race -run Test_Bully ./peer/tests/unit

test_avatar:
	go test -v -race -run Test_Avatar ./peer/tests/unit

lint:
	# Coding style static check.
	@go get -v honnef.co/go/tools/cmd/staticcheck
	@go mod tidy
	staticcheck ./...

vet:
	go vet ./...
