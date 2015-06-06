#!/bin/sh

echo "CMD $1"
echo "pkg $2"

for ARCH in debian/jessie debian/wheezy debian/squeeze ubuntu/vivid ubuntu/utopic ubuntu/trusty ubuntu/saucy ubuntu/precise
do
	package_cloud $1 "turbobytes/pulse/$ARCH" $2
done
