# go-prototype-rest-backend
A rest and search-indexer backend app build using golang.

![High Level arch diagram](./docs/images/go_rest_backend_app_arch.jpg "GO rest application")

## Tech Stack 
| Item                                       | version  | desc                                                |
| :----------------------------------------- | :------: | --------------------------------------------------: |
| golang                                     |   1.23   |                                                     |
| http router - chi                          |          | github.com/go-chi/chi/v5                            |
| redis client - go-redis                    |          | github.com/redis/go-redis/v9                        |
| kafka client - confluent-kafka-go          |          | github.com/confluentinc/confluent-kafka-go/v2/kafka |
| elasticsearch client - go-elasticsearch    |          | github.com/elastic/go-elasticsearch/v8              |
| godotenv                                   |  v1.5.1  | github.com/joho/godotenv                            |
| validator                                  |          | github.com/go-playground/validator/v10              |

## Project Structure
```
rest-backend
├── cmd
│   ├── api
│   │   ├── app
│   │   │   ├── app.go
│   │   │   └── envvar.go
│   │   └── main.go
│   ├── elasticsearch-indexer-kafka
│   │   ├── app
│   │   │   ├── app.go
│   │   │   └── envvar.go
│   │   └── main.go
│   ├── internal
│   │   └── common
│   │       ├── appcommon.go
│   │       ├── db.go
│   │       ├── elasticsearch.go
│   │       ├── env.go
│   │       ├── kafka.go
│   │       └── redis.go
│   ├── e2e_test.go
│   ├── e2e_testdata
│   │   ├── posts
│   │   │   └── test_case_prerequisites.json
│   │   └── user
│   │       ├── test_case_create_user.json
│   │       ├── test_case_delete_user.json
│   │       ├── test_case_get_user.json
│   │       └── test_case_update_user.json
│   └── migrate
│       └── migrations
├── devops
│   ├── docker
│   ├── helm
│   └── scripts
├── docs
├── internal
│   ├── api
│   │   ├── dto
│   │   │   ├── posts_dto.go
│   │   │   └── user_dto.go
│   │   ├── handler
│   │   │   ├── handlers.go
│   │   │   ├── posts_handler.go
│   │   │   ├── posts_handler_test.go
│   │   │   ├── users_handler.go
│   │   │   └── users_handler_test.go
│   │   ├── router.go
│   │   └── service
│   │       ├── mocks
│   │       │   ├── mock_posts_service.go
│   │       │   └── mock_user_service.go
│   │       ├── posts_service.go
│   │       ├── posts_service_test.go
│   │       ├── user_service.go
│   │       └── user_service_test.go
│   ├── common
│   │   ├── common.go
│   │   ├── domain
│   │   │   ├── model
│   │   │   │   ├── post.go
│   │   │   │   └── user.go
│   │   │   └── store
│   │   │       ├── db.go
│   │   │       ├── messagebroker.go
│   │   │       ├── mocks
│   │   │       ├── search.go
│   │   │       └── secret.go
│   │   └── error.go
│   ├── infrastructure
│   │   ├── cache
│   │   │   └── redis
│   │   │       ├── posts.go
│   │   │       ├── redis_test.go
│   │   │       ├── roles.go
│   │   │       └── users.go
│   │   ├── db
│   │   │   └── postgres
│   │   │       ├── postgres.go
│   │   │       ├── postgres_test.go
│   │   │       ├── posts.go
│   │   │       ├── roles.go
│   │   │       └── users.go
│   │   ├── messagebroker
│   │   │   └── kafka
│   │   │       ├── kafka_test.go
│   │   │       └── post.go
│   │   ├── search
│   │   │   └── elasticsearch
│   │   │       ├── elasticsearch_test.go
│   │   │       └── post.go
│   │   └── secret
│   │       └── envvarsecret
│   │           └── envvarsecret.go
│   └── testutils
│       ├── e2e_testutils.go
│       ├── testcontainers
│       │   ├── testcontainers_db.go
│       │   ├── testcontainers_elasticsearch.go
│       │   ├── testcontainers_kafka.go
│       │   └── testcontainers_redis.go
│       ├── testfixtures_http.go
│       └── testutils.go
├── docker-compose-dev-instance.yml
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── shell.nix
```
## Project setup 

### Project Prerequisite 
* golang
* docker instance - required for running testcontainers related test. check shell.nix to configure remote docker
* delve - [optional] for debugging go projects
* air - [optional] for hot/live reloading go projects

### Project Initial setup

#### Init the module 
```
go mod init github.com/kannancmohan/go-prototype-rest-backend
```

#### [optional] Init air for hot reloading
```
air init
```
adjust the generated '.air.toml' file to accommodate project specif changes

### Project Build & Execution

#### Project environment variables 

* For development environment:

     The env variables can be defined in .envrc file. The direnv tool will automatically load the env variables from .envrc file
     
     if you update the .envrc file on the fly, use command "direnv reload" to reload the env variables

#### API App Build & Execution

##### Build API App
```
make build
or
go build ./...
```

##### Run API App
```
make run-api
or
go run cmd/api/*.go
```

##### Run API App Test
```
make test
or
go test -v ./...
```

##### Run API App Test (skip testcontainers related test)
```
make test-skip-docker-tests
or
go test -v -tags skip_docker_tests ./...
```

#### Search indexer App Build & Execution

##### Build search indexer App
```
make build
or
go build ./...
```

##### Run search indexer App
```
make run-indexer
or
go run cmd/elasticsearch-indexer-kafka/*.go
```

#### Project DB migration
##### To add new migration file

```
make migration-create user_table
```
##### To migrate db

```
make migration-up
```

##### To revert db migration

```
make migration-down
```

## Additional 

"accept interfaces and return concrete types(struct)" 
