# go-prototype-rest-backend
A rest backend app build using golang.

## Tech Stack 

golang : 1.22
chi http router : github.com/go-chi/chi/v5


## Project Structure
```
rest-backend/
├── cmd/
│   ├── api-server/
│   │   ├── main.go
│   │   └── ...
│   └── db-migration/
│       ├── main.go
│       └── migration/
├── internal/
│   ├── common/
│   │   ├── config/
│   │   │   └── env.go
│   │   └── db/
│   │   │   └── db.go
│   ├── api-server/
│   │   ├── handler/
│   │   │   ├── user_handler.go
│   │   │   └── ...
│   │   ├── service/
│   │   └── router.go
│   └── db-migration/
│       ├── migration/
│       └── ...
├── infra/
│   ├── Dockerfile
│   └── k8s/
│       └── ...
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── shell.nix #to spawn a shell environment with specific tools and dependencies

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