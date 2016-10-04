export GOPATH=/home/martin/Projects/IceMenu/library

export GOROOT=/opt/go-1.7.1/
export PATH=$GOROOT/bin:$PATH

go build -ldflags "-s -w" -o filebrowser ./ && cp filebrowser ~/bin/

