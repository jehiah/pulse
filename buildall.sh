#!/bin/sh

VERSION="1.0.30"
LDFLAGS="-X main.version $VERSION"
set -o xtrace

for ARCH in amd64 arm 386
do
	for OS in linux darwin windows
	do
		if [ "$ARCH" = "amd64" ] || [ "$OS" = "linux" ]; then #process arm only for linux
			echo "$OS-$ARCH"
			GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o minion minion.go
			tar -czf "minion.$OS.$ARCH.tar.gz" minion
			sha256sum "minion.$OS.$ARCH.tar.gz" >  "minion.$OS.$ARCH.tar.gz.sha256sum"
			rm minion
		fi
	done
done
/usr/local/bin/s3cmd --acl-public put  *.tar.gz* "s3://tb-minion/"
rm *.tar.gz*
echo $VERSION > latest
s3cmd --acl-public put latest "s3://tb-minion/"
rm latest
