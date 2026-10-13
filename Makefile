.PHONY: up down run run-admin run-home run-user run-gateway migration migration-user proto build test vet tidy clean

up:
	docker compose up -d

down:
	docker compose down

run: ## 一键启动四服务（etcd 需先 make up；Ctrl+C 全部优雅退出）
	@go run ./app/user/cmd/server -conf config/user/local.yml & \
	go run ./app/admin/cmd/server -conf config/admin/local.yml & \
	go run ./app/home/cmd/server -conf config/home/local.yml & \
	go run ./app/gateway/cmd/server -conf config/gateway/local.yml & \
	wait

run-gateway:
	go run ./app/gateway/cmd/server -conf config/gateway/local.yml
	@go run ./app/user/cmd/server -conf config/user/local.yml & 	go run ./app/admin/cmd/server -conf config/admin/local.yml & 	go run ./app/home/cmd/server -conf config/home/local.yml & 	wait

run-admin:
	go run ./app/admin/cmd/server -conf config/admin/local.yml

run-home:
	go run ./app/home/cmd/server -conf config/home/local.yml

run-user:
	go run ./app/user/cmd/server -conf config/user/local.yml

migration-user:
	go run ./app/user/cmd/migration -conf config/user/local.yml

proto:
	protoc --go_out=. --go_opt=module=do-template-app \
		--go-grpc_out=. --go-grpc_opt=module=do-template-app \
		api/proto/userv1/user.proto

migration:
	go run ./app/admin/cmd/migration -conf config/admin/local.yml

build:
	CGO_ENABLED=0 go build -o bin/admin ./app/admin/cmd/server
	CGO_ENABLED=0 go build -o bin/home ./app/home/cmd/server
	CGO_ENABLED=0 go build -o bin/user ./app/user/cmd/server
	CGO_ENABLED=0 go build -o bin/gateway ./app/gateway/cmd/server

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin storage/do-template-app.db storage/do-template-app.db-*
