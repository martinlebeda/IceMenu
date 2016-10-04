# IceMenu

Menu generator for browse filesystem for icewm.

# Usage

You can use `--help`:

```
Usage of filebrowser:
  -annex string
    	open script for git annex managed files
  -avfs string
    	mountpoint of avfsdirectory
  -dir string
    	directories for menu, : is separator (default "$HOME")
  -norecur
    	prevent recursive directories 
  -open string
    	open script for files (default "xdg-open")
  -suffix string
    	include only suffixes, default is all, : is separator
```

example usage in `~/.icewm/menu` (if `filebrowser` is in `$PATH`):
```
menuprogreload "Home" folder 10 /home/martin/bin/filebrowser -annex /home/martin/bin/getAndOpen.sh -avfs /home/martin/.avfs -dir /home/martin
menuprogreload Podcasts podcast.png 0 filebrowser -dir /home/martin/Mobile/ExtraPodcast:/home/martin/Video/other:/home/martin/Downloads -suffix .avi:.mp4:.flv:.webm:.mkv:.ts:.mp3 -open /home/martin/bin/playAndDelete.sh -norecur
```

# Configuration
TODO 

# Future plans 

in unspecified future/maybe if I get feel or I need it:

- work with submenu generated at once with main
- insert param for switch hiden files
- support for blackbox wm menu format
- some other functionality...
