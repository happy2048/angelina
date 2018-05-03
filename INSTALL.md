Install:
	old=$(echo $GOPATH) && export GOPATH=$(pwd) && make && export GOPATH=$old
