#!/bin/sh

VERSION="1.0.59"
LDFLAGS="-X main.version=$VERSION"
KEY=$1 #The key to sign the package with

if [ "$KEY" = "" ]; then
	echo "Must provide gpg signing key"
	exit 1
fi

set -o xtrace

for ARCH in amd64 arm 386
do
	for OS in linux darwin windows freebsd android
	do
		if [ "$ARCH" = "amd64" ] || [ "$OS" = "linux" ] || [ "$OS" = "freebsd" ] || [ "$OS" = "android" ]; then #process arm only for linux and freebsd
			echo "$OS-$ARCH"
			if [ "$OS" = "windows" ]; then
				#Windows binary is exe
				GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o minion.exe minion.go
				tar -czf "minion.$OS.$ARCH.tar.gz" minion.exe
			else
				if [ "$OS" = "android" ]; then
					if [ "$ARCH" = "arm" ]; then
						GOMOBILE="/home/sajal/go/pkg/gomobile" GOOS=android GOARCH=arm CC=$GOMOBILE/android-ndk-r10e/arm/bin/arm-linux-androideabi-gcc CXX=$GOMOBILE/android-ndk-r10e/arm/bin/arm-linux-androideabi-g++ CGO_ENABLED=1 GOARM=7 go build -p=8 -pkgdir=$GOMOBILE/pkg_android_arm -tags="" -ldflags="$LDFLAGS -extldflags=-pie" -o minion minion.go
						tar -czf "minion.$OS.$ARCH.tar.gz" minion
					fi
				else
					GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o minion minion.go
					tar -czf "minion.$OS.$ARCH.tar.gz" minion
				fi
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
