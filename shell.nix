let
    nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.05";
    pkgs = import nixpkgs { config = {}; overlays = []; };
    isMacOS = pkgs.stdenv.hostPlatform.system == "darwin";
    remoteDockerHost = "ssh://ubuntu@192.168.0.30"; ## set this if you want to use remote docker, else set it to ""
in

pkgs.mkShellNoCC {
    packages = with pkgs; [
        pkgs.tree # optional
        ## added for golang
        pkgs.go_1_22
        pkgs.go-migrate # for using go migration cli from https://github.com/golang-migrate/migrate
        pkgs.delve # debugger for Go
        pkgs.air # hot reload for Go
        pkgs.clang # added for vs-code Go extension to work
        pkgs.golangci-lint
        ## direnv for project/shell specific env variables(see .envrc file)
        pkgs.direnv
        ## added golang testing
        pkgs.docker
    ] ++ (if pkgs.stdenv.isDarwin then [ 
        pkgs.darwin.iproute2mac pkgs.darwin.apple_sdk.frameworks.CoreFoundation 
    ] else []);

    shellHook = ''
        ## Add environment variables
        #export GOROOT=${pkgs.go}/share/go
        #export GOPATH=$PWD

        ## Add command alias
        #alias k="kubectl"

        ### direnv ###
        eval "$(direnv hook bash)"
        if [ -f .envrc ]; then
            export DIRENV_LOG_FORMAT="" #for disabling log output from direnv
            echo ".envrc found. Allowing direnv..."
            direnv allow .
        fi

        ### Conditional set CGO_LDFLAGS ###
        if $isMacOS; then
            export CGO_CFLAGS="-mmacosx-version-min=13.0"
            export CGO_LDFLAGS="-mmacosx-version-min=13.0"
        fi

        if [ "${remoteDockerHost}" != "" ]; then
            echo "Using remote Docker and setting ssh for the same"
            export DOCKER_HOST="${remoteDockerHost}"
            ## Start SSH agent if not already running
            if [ -z "$SSH_AGENT_PID" ]; then
                eval $(ssh-agent -s)
                echo "SSH agent started with PID $SSH_AGENT_PID"
            fi
            ## Add host ssh keys
            if $isMacOS; then
                ssh-add --apple-use-keychain ~/.ssh/id_ed25519
            else
                ssh-add ~/.ssh/id_ed25519
            fi
        else
            echo "Using local Docker"
        fi
        # Ensure Docker is running(required for testcontainers etc)
        if ! docker info > /dev/null 2>&1; then
            echo "Docker is not running. Please check the Docker daemon."
        fi

    '';
}
