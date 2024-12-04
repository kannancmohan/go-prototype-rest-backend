# go-prototype-rest-backend
A rest backend app build using golang 

## Tech Stack 
golang : 1.22

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

### Prerequisite 
* golang (preinstalled via nix)

### Initial setup

#### Init the module 
```
go mod init github.com/kannancmohan/go-prototype-rest-backend
```