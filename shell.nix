let
    nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.05";
    pkgs = import nixpkgs { config = {}; overlays = []; };
in

pkgs.mkShellNoCC {
    packages = with pkgs; [
        ## added for golang
        pkgs.go_1_22
        pkgs.go-migrate # for using go migration cli from https://github.com/golang-migrate/migrate
        pkgs.delve # debugger for Go
        pkgs.air # hot reload for Go
        pkgs.clang # added for vs-code Go extension to work
        pkgs.golangci-lint
    ] ++ (if pkgs.stdenv.isDarwin then [ 
        pkgs.darwin.iproute2mac pkgs.darwin.apple_sdk.frameworks.CoreFoundation 
    ] else []);

    shellHook = ''
        ## Add environment variables
        export GOROOT=${pkgs.go}/share/go
        export GOPATH=$PWD

        ## Add command alias
        #alias k="kubectl"
    '';
}
