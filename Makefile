#.PHONY: docker
#docker:
#	@del webook || true
#	@go mod tidy
#	@go env -w GOOS=linux
#	@go env -w GOARCH=arm
#	@go build -tags=k8s -o webook .
#	@docker build -t nanxu/webook-live:v0.0.1 .

.PHONY: mock
mock:
	@mockgen -source ./internal/service/user.go -package=svcmocks -destination=internal/service/mocks/user.mock.go
	@mockgen -source ./internal/service/code.go -package=svcmocks -destination=internal/service/mocks/code.mock.go
	@mockgen -source ./internal/repository/user.go -package=repomocks -destination=internal/repository/mocks/user.mock.go
	@mockgen -source ./internal/repository/code.go -package=repomocks -destination=internal/repository/mocks/code.mock.go
	@mockgen -source ./internal/repository/dao/user.go -package=daomocks -destination=internal/repository/dao/mocks/user.mock.go
	@mockgen -source ./internal/repository/cache/user.go -package=cachemocks -destination=internal/repository/cache/mocks/user.mock.go
	@mockgen -source ./internal/service/sms/types.go -package=smsmocks -destination=internal/service/sms/mocks/sms.mock.go
	@mockgen -source ./pkg/ratelimit/types.go -package=limitermocks -destination=pkg/ratelimit/mocks/ratelimit.mock.go
