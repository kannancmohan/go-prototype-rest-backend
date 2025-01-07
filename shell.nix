let
    nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.05";
    pkgs = import nixpkgs { config = {}; overlays = []; };
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
        if [ "$(uname)" = "Darwin" ]; then
            export CGO_CFLAGS="-mmacosx-version-min=13.0"
            export CGO_LDFLAGS="-mmacosx-version-min=13.0"
            #echo "CGO_LDFLAGS set for macOS (Darwin)"
        fi
    '';
}
