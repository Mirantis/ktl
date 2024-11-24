Hide
Type "alias diff='delta --side-by-side --diff-highlight --file-style=omit --hunk-header-style=omit'"
Type ";clear"
Enter
Type "diff <(echo '# --- no groups ---'; cd all/components; ls) <(echo '# --- groups ---'; cd groups/components; ls) | less -r"
Show
Enter
Sleep 2.5s
Hide
Type q
Show
