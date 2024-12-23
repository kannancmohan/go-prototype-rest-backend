# go-prototype-rest-backend
A rest backend app build using golang.

## Tech Stack 

golang : 1.22
chi http router : github.com/go-chi/chi/v5


## Project Structure
```
rest-backend
├── cmd
│   ├── api
│   │   ├── envvar.go
│   │   ├── main.go
│   │   ├── memcache.go
│   │   └── postgres.go
│   ├── internal
│   │   └── common
│   │       ├── db.go
│   │       └── env.go
│   └── migrate
│       └── migrations
├── devops
│   ├── docker
│   │   └── Dockerfile
│   ├── helm
│   └── scripts
├── internal
│   ├── api
│   │   ├── common
│   │   │   └── error.go
│   │   ├── config
│   │   │   └── config.go
│   │   ├── domain
│   │   │   └── model
│   │   │       ├── post.go
│   │   │       └── user.go
│   │   ├── dto
│   │   │   └── user.go
│   │   ├── handler
│   │   │   ├── handlers.go
│   │   │   ├── posts.go
│   │   │   └── users.go
│   │   ├── router.go
│   │   ├── service
│   │   │   ├── posts.go
│   │   │   └── user.go
│   │   └── store
│   │       └── store.go
│   └── infrastructure
│       ├── envvar
│       ├── memcache
│       │   └── postgres
│       │       ├── posts.go
│       │       ├── roles.go
│       │       └── users.go
│       └── postgres
│           ├── posts.go
│           ├── roles.go
│           └── users.go
├── Makefile
├── README.md
├── docker-compose-postgres.yml
├── go.mod
├── go.sum
└── shell.nix
```
## Project setup 

### Project Prerequisite 
* golang
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