Hide
Type "ls -T ./test"
Enter
Show
Sleep 1.5s

Hide
Type "alias diff='delta --side-by-side --diff-highlight --file-style=omit --hunk-header-style=omit'"
Type ";clear"
Enter
Type "diff <(echo '# --- dev ---'; cd dev; ls) <(echo '# --- test ---'; cd test; ls) | less -r"
Show
Enter
Sleep 5s
Hide
Type q
Show

Hide
Type "alias diff='delta --side-by-side --diff-highlight --file-style=ul\ bold\ yellow --hunk-header-style=omit'"
Type ";clear"
Enter
Type "diff dev/myapp/myapp-deployment.yaml test/myapp/myapp-deployment.yaml | less -r"
Show
Enter
Sleep 5s
Hide
Type q
Type clear
Enter
Show
