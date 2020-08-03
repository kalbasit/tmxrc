{ pkgs ? import (import ./nix/pkgs.nix) {}, version ? "dev" }:

with pkgs;

buildGoModule rec {
  inherit version;

  pname = "swm";

  src = nix-gitignore.gitignoreSource [ ".git" ".envrc" ".travis.yml" ".gitignore" ] ./.;

  vendorSha256 = null;

  buildFlagsArray = [ "-ldflags=" "-X=main.version=${version}" ];

  nativeBuildInputs = [ fzf git tmux procps installShellFiles ];

  subPackages = [ "." ];

  postInstall = ''
    for shell in bash zsh fish; do
      $out/bin/swm auto-complete $shell > swm.$shell
      installShellCompletion swm.$shell
    done

    $out/bin/swm gen-doc man --path ./man
    installManPage man/*.7
  '';

  doCheck = true;
  preCheck = ''
    export HOME=$NIX_BUILD_TOP/home
    mkdir -p $HOME

    git config --global user.email "nix-test@example.com"
    git config --global user.name "Nix Test"
  '';

  meta = with lib; {
    homepage = "https://github.com/kalbasit/swm";
    description = "swm (Story-based Workflow Manager) is a Tmux session manager specifically designed for Story-based development workflow";
    license = licenses.mit;
    maintainers = [ maintainers.kalbasit ];
  };
}