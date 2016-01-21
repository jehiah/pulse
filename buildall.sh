#!/bin/sh

VERSION="1.0.57"
LDFLAGS="-X main.version=$VERSION"
KEY=$1 #The key to sign the package with

if [ "$KEY" = "" ]; then
	echo "Must provide gpg signing key"
	exit 1
fi

set -o xtrace

for ARCH in amd64 arm 386
do
	for OS in linux darwin windows freebsd
	do
		if [ "$ARCH" = "amd64" ] || [ "$OS" = "linux" ] || [ "$OS" = "freebsd" ]; then #process arm only for linux and freebsd
			echo "$OS-$ARCH"
			if [ "$OS" = "windows" ]; then
				#Windows binary is exe
				GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o minion.exe minion.go
				tar -czf "minion.$OS.$ARCH.tar.gz" minion.exe
			else
				GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o minion minion.go
				tar -czf "minion.$OS.$ARCH.tar.gz" minion
			fi
			sha256sum "minion.$OS.$ARCH.tar.gz" >  "minion.$OS.$ARCH.tar.gz.sha256sum"
			gpg --default-key $KEY --output "minion.$OS.$ARCH.tar.gz.sig" --detach-sig "minion.$OS.$ARCH.tar.gz"
			rm minion
			rm minion.exe
		fi
	done
done
/usr/local/bin/s3cmd --acl-public put  *.tar.gz* "s3://tb-minion/"
rm *.tar.gz*
echo $VERSION > latest
s3cmd --acl-public put latest "s3://tb-minion/"
rm latest
