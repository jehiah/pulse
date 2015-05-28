# TurboBytes Pulse

Pulse is a tool to run network diagnostics in a distributed manner. It is made up of 2 components.

- CNC : This is the Command & Control server. Users make http requests to it describing the test they want to run. CNC then runs it across all minions, gathers the response and then returns them to the user.
Dependencies : mongodb (I might replace it with something lighter, or make it optional) 

- minion - This is the agent that runs at places where you want to debug from. It makes a TLS connection to CNC and waits for incoming test requests to be executed.
Dependencies: mtr command (ubuntu: apt-get install mtr-tiny)

## Build instructions

	go get github.com/turbobytes/pulse
	cd $GOPATH/src/github.com/turbobytes/pulse
	go build cnc.go
	go build minion.go

You can build these for any target supported by Go by manipulating `GOOS` and `GOARCH`.  gccgo is not supported currently because it uses older Go versions. You might need to adapt the code for gccgo. We had success in [running it on MIPS](http://www.sajalkayan.com/post/golang-openwrt-mips.html) as proof of concept.

It is important that the CNC and minion are from the same release. If you are updating one of them then its crucial to update the other to avoid unexpected behaviour. They share data structures.

## TLS PKI

CNC and minion use TLS to communicate with each other. Use your own CA to sign the certificates and minion and CNC trusts only this CA. The TLS setup was inspired by [this blogpost](http://www.hydrogen18.com/blog/your-own-pki-tls-golang.html).

#### Install EasyRSA

	git clone https://github.com/OpenVPN/easy-rsa.git example-ca
	chmod 700 example-ca
	cd example-ca
	rm -rf .git


#### Create CA

	./easyrsa init-pki

Its important that you set a passphrase for your CA's private key

#### Create server cert

	./easyrsa build-server-full localhost nopass

Replace localhost with the hostname of the server that runs the CNC.

#### Create minion certificate

	./easyrsa build-client-full 'client0' nopass

Create one certificate for each minion instance. Replace 'client0' with some descriptive name. This is whats shown in the ui/api to indicate which agent ran the test.

## Running Pulse

#### CNC

The CNC needs mongodb running on localhost. This requirement will be removed in future releases. Mongo is only used for storing metadata about minions.

usage : `./cnc -ca="/path/to/ca.crt" -crt="/path/to/server.crt" -key="/path/to/server.key"`

Note: `server.crt` and `server.key` is the certificate/key generated using the `build-server-full` command.

Its important that all minions can reach port 7777 on the server, and all users can reach port 7778.

#### minion

usage : `./minion -ca="/path/to/ca.crt" -crt="/path/to/minion.crt" -key="/path/to/minion.key"`

Note: `minion.crt` and `minion.key` is the certificate/key generated using the `build-client-full` command.

Use one client certificate exclusive to one minion.

## Using Pulse

visit http://[cnc hostname]/agents/ for a listing of online agents.

#### DNS test

TODO

#### HTTP test

TODO

#### mtr/traceroute

TODO

## Create new test types

TODO