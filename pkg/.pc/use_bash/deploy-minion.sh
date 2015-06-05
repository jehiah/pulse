#!/bin/sh

#Run this with OS and ARCH defined in enviornment....
#OS="linux" ARCH="amd64" ./deploy-minion.sh  2>&1 | logger -t minion &

#Check if minion is latest or not...

main_function {
	#Some autodetction for OS...
	if [ "$OS" = "" ]; then
		unamestr=`uname`
		if [ "$unamestr" = 'Linux' ]; then
		   OS='linux'
		fi
	fi

	#Some autodetction for ARCH...
	if [ "$ARCH" = "" ]; then
		unamestr=`uname -m`
		#Matches my laptop
		if [ "$unamestr" = 'x86_64' ]; then
		   ARCH='amd64'
		fi
		#Matches rpi debian
		if [ "$unamestr" = 'armv6l' ]; then
		   ARCH='arm'
		fi
		#Matches online labs c4
		if [ "$unamestr" = 'armv7l' ]; then
		   ARCH='arm'
		fi
	fi

	if [ "$OS" = "" ]; then
		echo "Must provide enviornment variable OS"
		exit 1
	fi


	if [ "$ARCH" = "" ]; then
		echo "Must provide enviornment variable ARCH"
		exit 1
	fi

	TARFILE="minion.$OS.$ARCH.tar.gz"
	SHAFILE="minion.$OS.$ARCH.tar.gz.sha256sum"
	BASEURL="https://s3.amazonaws.com/tb-minion/"

	echo "$TARFILE $SHAFILE"
	#set -o xtrace
	while :
	do
		if [ ! -f current ]; then
		    echo "none" > current
		fi

		if [ ! -f $TARFILE ]; then
		    echo "none" > current
		fi

		if [ ! -f minion ]; then
		    echo "none" > current
		fi

		curl -so latest "${BASEURL}latest"

		diff --brief current latest >/dev/null
		comp_value=$?

		if [ $comp_value -eq 1 ]
		then
			#Current did not match latest
		    echo "need to upgrade..."
		    curl -so "$TARFILE" "$BASEURL$TARFILE"
		    curl -so "$SHAFILE" "$BASEURL$SHAFILE"
		    sha256sum -c "$SHAFILE" > /dev/null
		    if [ $? -eq 0 ]
		    then
		    	echo "Successfully downloaded"
		    	tar -xf "$TARFILE"
		    	cp latest current
		    fi
		else
		    echo "no need to upgrade..."
		fi

		./minion -cnc="distdns.turbobytes.com:7777" $EXTRAARGS
		sleep 60 #rest for a minute... Avoid crash loop...
	done
}

main_function 2>&1 | logger -t minion