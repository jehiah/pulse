#!/bin/bash

#Run this with OS and ARCH defined in enviornment....
#OS="linux" ARCH="amd64" ./deploy-minion.sh  2>&1 | logger -t minion &

#Install GPG key if not present
gpg --list-keys 24F6D50F
HASGPG=$?

if [ $HASGPG -gt 0 ]
then
	#Key is not present.
	echo "Installing Public key"
	gpg --import <<EOF
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1.4.12 (GNU/Linux)

mQMuBFV7Ar0RCADaT4Uzyq7ZHIkR5/WqbcLTkN9jxj4kp5XKzNQt9nrIbVw8nRt6
r3+2FFMPdLnSCqDpuz5X5pUnUlny+T5fgx0/OCJrz4J3iUgMftxc1TYN80rs5HuM
ClZqovw2T4VOvS+jRqJErzMUcAPIY4EPCNxQTWpcnjzQfrw5aLgAZ80wjZr7gpUf
dC2PgkW3QZtCtkTD8LB59fjeaVnRuWlQ7CXKX+MNxLGHD3BkZxHV7NBoc0TTJiHr
QGwS5/Ghiqbnm2julWmZKShB6s97ZDBfLCD4iSPbOZyKJIYlcGwhp3boqzL+714Q
2n16bZcEsnKI/Hle4tOKjJLk67rM7hM5oEdXAQCKAcvkNuAsTmEBg7PTa3iFfKxE
NDIS6A5r3qLWLISqwQf+NFJMa29AcTRSQNC587qjuxR/u2owBUtdkzyl0fIYeBXO
+LFTm9gJRXiNBsFI0A/qnyyAXHL4Vkf79hz6JW+jnFglvpXE0RebPSPLeWOdn3Bb
Mid8mm1iFagjstITqXy/RdzjFaoeTsl40JlyYiGPU2lvfMKWimVQ97E2Gn00kKrZ
HLvCHjANGY0nMnyUFroVdO9yZ3tM3dOFfL+TV/MnnaokFFOxbd7Gxq27ZcYIs7kO
alnroHHsWCCemidF0TAzJexF1AXAVfMMacxeJD3yPX6SUqPbloDf6WRPfAhjPIjw
WeRb3dhcd+/ct21gP5pG8U1pPJ+/yCGiVKn5MF8cwggA1xLmX1Xx4Z1Ncu+V6YHy
ZHbVAb3vtBZnL+hdYoJxpDoV7ML0SDX7ZsMXQ65eD0NHSCJehcK1jkYDwMvf8mi8
pQL3+veXsGh41uHPl9sFGHpZZCvfvggcDLr5Pa0gQuLOpUiXctUmw60B2Xvcp0js
6R98TKaeIyOJMVp3OTO95JaVZFxpYCJqzs5GFBroMpPYCIWn0vNLp3HOx2R59Y3Y
rfcD727Z0aG1MEnqWmShutTHXG/hm2no/nyDYxSWLq17ZQjhPO5pF6qcoy7zhhrh
6uJrfPLT41D2/HH4XDHKPBYxdyEBWt4EAC0bWcgSBnM8TcfdmfraFC+2DX9ZM7Q+
KbRFU2FqYWwgS2F5YW4gKFNpZ25pbmcga2V5IGZvciBUdXJib0J5dGVzIFB1bHNl
KSA8c2FqYWxAdHVyYm9ieXRlcy5jb20+iIEEExEIACkFAlV7Ar0CGwMFCQlmAYAH
CwkIBwMCAQYVCAIJCgsEFgIDAQIeAQIXgAAKCRC3o80eJPbVD/NbAQCJcBRISrWH
MC04vRPS/XLVTjJhLOApy0uMmfvbEZr6dAD9ESndQ71KPKQ3I/ikKJOEbBx9Kxzl
56OObA0/fiMHJns=
=x/VJ
-----END PGP PUBLIC KEY BLOCK-----
EOF
	gpg --list-keys 24F6D50F
	HASGPG=$?
	#If key is present and gpg is working fine, then HASGPG should be zero
fi

if [ $HASGPG -eq 0 ]
then
	echo "We will use GPG for veryfying tar downloads and not sha256sum"
fi

#Check if minion is latest or not...

function main_function
{
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
		#Match 386
		if [ "$unamestr" = 'i686' ]; then
		   ARCH='386'
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
	GPGFILE="minion.$OS.$ARCH.tar.gz.sig"
	BASEURL="https://tb-minion.turbobytes.net/"

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
		    curl -so "$GPGFILE" "$BASEURL$GPGFILE"
			if [ $HASGPG -eq 0 ]
			then
				#Validate using gpg
				gpg --verify "$GPGFILE"
			else
				#Validate sha256sum as fallback...
		    	sha256sum -c "$SHAFILE" > /dev/null
			fi
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