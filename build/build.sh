#!/bin/sh
self=`readlink "$0"`
if [ -z "$self" ]; then
	self=$0
fi
scriptname=`basename "$self"`
scriptdir=${self%$scriptname}

cd $scriptdir
scriptdir=`pwd`

cd ..

git_tag=`git describe --exact-match HEAD 2>/dev/null`
git_hash=`git log --pretty=format:'%h' -n 1`
go build -ldflags "-X main.gitTag=$git_tag -X main.gitHash=$git_hash"
