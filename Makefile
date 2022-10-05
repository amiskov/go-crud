run:
	go run ./...
fmt:
	echo "goimports:" && goimports -l -local "crud" -w . && \
	echo "gofumpt:" && gofumpt -l -w .
lint:
	golangci-lint run ./...
test:
	go test ./... -run=TestGetUserPosts -v
mock:
	mockgen -source=pkg/user/api/handlers.go -destination=pkg/user/api/handlers_mock.go -package=api UserRepo SessionManager
mockmongo:
	mockgen -source=pkg/post/mongo_interfaces.go -destination=pkg/post/mongo_mock.go -package=post IMongoDB
mockposts:
	mockgen -source=pkg/post/api/repo.go -destination=pkg/post/api/repo_mock.go -package=post UserRepo SessionManager
mocktest:
	make mock && make test
coverhtml:
	go test ./... -coverprofile=cover.out && go tool cover -html=cover.out
coverfunc:
	go test ./... -coverprofile=cover.out.tmp; \
	cat cover.out.tmp | grep -v "_mock.go" > cover.out; \
	go tool cover -func=cover.out