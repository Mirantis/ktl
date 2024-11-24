# NOTE: playback speed bug on mac: https://github.com/charmbracelet/vhs/issues/88
Set Width 1920
Set Height 1080
Set Margin 60
Set MarginFill "#3d348b"
Set WindowBar Colorful
Set BorderRadius 16
Set FontSize 30
Set FontFamily "Terminess Nerd Font Mono"
Set Shell zsh
Set Framerate 15
Set TypingSpeed 0.05

Output data/demo.mp4

Hide
Type "cd data"
Type ";alias rekustomize=true"
Type ";alias kubectl=true"
Type ";alias ls='eza --icons=always --color=always'"
Type ";alias view='bat --paging=always --color=always'"
Type ";clear"
Enter
Show

Source "01-title.tape"
Source "02-export-dev.tape"
Source "03-show-dev.tape"
Source "04-export-test.tape"
Source "05-show-test.tape"
Source "06-export-all.tape"
Source "07-show-all.tape"
Source "08-export-groups.tape"
Source "09-show-groups.tape"
Source "10-fin.tape"
