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
        pkgs.delve # debugger for Go
        pkgs.air # hot reload for Go
        pkgs.clang # added for vscode Go extension to work
        pkgs.cyrus_sasl # added for vscode Go extension to work
        pkgs.direnv ## direnv for project/shell specific env variables(see .envrc file)
        ## tools for the go project
        pkgs.golangci-lint
        pkgs.go-migrate # for using go migration cli from https://github.com/golang-migrate/migrate
        pkgs.mockgen # for using mockgen cli from https://github.com/uber-go/mock 
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
            if [[ "${remoteDockerHost}" == ssh://* ]]; then
                echo "Using remote Docker and setting ssh for the same"
                
                ## Extract username and IP from remoteDockerHost
                sshUser=$(echo "${remoteDockerHost}" | sed -E 's|ssh://([^@]+)@.*|\1|')
                sshIp=$(echo "${remoteDockerHost}" | sed -E 's|ssh://[^@]+@(.*)|\1|')
                
                ## Check if SSH access is possible to remotehost
                if ssh -o BatchMode=yes -o ConnectTimeout=5 -q $sshUser@$sshIp exit; then
                    echo "SSH access to $sshIp@$sshIp verified successfully."
                else
                    echo "Error: Unable to SSH into $username@$ip. Please check your SSH configuration and access permissions."
                    exit 1
                fi

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

                ## SSH tunneling to remote host for using its docker instance 
                rm -f /tmp/remote-docker-gotest.sock # remove remote-docker-gotest.sock if it already exists 
                ssh -f -N -L /tmp/remote-docker-gotest.sock:/var/run/docker.sock $sshUser@$sshIp # do the ssh tunneling
                
                ## Set Docker and testcontainers env variables
                export DOCKER_HOST=unix:///tmp/remote-docker-gotest.sock
                export TESTCONTAINERS_HOST_OVERRIDE=$sshIp
                export TESTCONTAINERS_RYUK_DISABLED=true # disabling ryuk for now. Reason: ryuk port binding is failing for some reason
                
            else
                echo "Invalid remoteDockerHost format. Expected 'ssh://<username>@<ip>'."
                exit 1
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
