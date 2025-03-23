{
  projectRootFile = "treefmt.nix";

  programs.gofmt.enable = true;
  programs.mdformat.enable = true;
  programs.nixfmt.enable = true;
  programs.black.enable = true;
}
