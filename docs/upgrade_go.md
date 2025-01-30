
## Golang version upgrade
### For nix-shell
#### Check nix for available golang package
Check https://lazamar.co.uk/nix-versions/?channel=nixpkgs-24.05-darwin&package=go for available go lang versions for your operating system and architecture

#### Update the golang package version in shell.nix
eg pkgs.go_1_23

### For sudo rm -rf /usr/local/go

#### Uninstall old go version 
use command "which go" to get current installed location 
```
$ sudo rm -rf /usr/local/go
```

#### Install new go version 
Visit the official Go downloads page: https://golang.org/dl/ and download the appropriate package for your operating system and architecture
```
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
```

## [Optional] Check dependencies compatibility for a go version
check if the current dependencies of a Go project are compatible with a specific version of Go
```
go mod tidy -go=1.23
```
This command will 

    Update the go.mod file to reflect the specified Go version
    Check if all dependencies are compatible with that version.
    Remove any dependencies that are not needed or incompatible

## Projects dependencies(go packages) upgrade

### [Optional] Check Dependency Compatibility
```
go list -u -m all
```
The result will show something like 
```
github.com/kannancmohan/go-prototype-rest-backend
cloud.google.com/go v0.112.1 [v0.118.0]
```
Where the first line is your projects main module. And below it, each dependencies are listed with its name, current version and on [] you see new version if available

### Upgrade all dependencies 
```
go get -u ./...
go mod tidy
```

### Upgrade individual dependency
```
go get github.com/some/dependency@v1.3.0
go mod tidy
```