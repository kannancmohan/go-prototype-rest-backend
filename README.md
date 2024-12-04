# go-prototype-rest-backend
A rest backend app build using golang 

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
│   │   ├── configs/
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

#### Project 