[![Build Status](https://travis-ci.org/turbobytes/pulse.png?branch=master)](https://travis-ci.org/turbobytes/pulse)

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

Its important that system times are correct. If not then TLS might not work correctly.

#### CNC

The CNC needs mongodb running on localhost. This requirement will be removed in future releases. Mongo is only used for storing metadata about minions.

usage : `./cnc -ca="/path/to/ca.crt" -crt="/path/to/server.crt" -key="/path/to/server.key"`

Note: `server.crt` and `server.key` is the certificate/key generated using the `build-server-full` command.

Its important that all minions can reach port 7777 on the server, and all users can reach port 7778.

#### minion

usage : `./minion -ca="/path/to/ca.crt" -crt="/path/to/minion.crt" -key="/path/to/minion.key" -cnc="cnc.host.name:7777"`

Note: `minion.crt` and `minion.key` is the certificate/key generated using the `build-client-full` command. `cnc.host.name` is the hostname of the CNC

Use one client certificate exclusive to one minion.

## Using Pulse

Visit http://cnc.host.name:7778/agents/ for a listing of currently online agents.

http://cnc.host.name:7778/ contains a rough demo UI to run tests.

#### DNS test

API endpoint: /dns/
Method: POST
Payload: Json object

example :-

	{
		"Host": "example.com",
		"QType": 1,
		"Targets": ["8.8.8.8", "8.8.4.4"]
	}

`Host` : The hostname we want to resolve
`QType` : Dns [query type](http://en.wikipedia.org/wiki/List_of_DNS_record_types#Resource_records)
`Targets` : The nameservers we want to query

#### HTTP test

API endpoint: /curl/
Method: POST
Payload: Json object

example :-

	{
		"Path": "/foo/bar.jpg",
		"Endpoint": "example.com",
		"Host": "foobar.com",
		"Ssl": false
	}

`Path` : The URI to test
`Endpoint` : The server to connect to.
`Host` : The contents of the Host header. If blank then endpoint's value is used here.
`Ssl` : Weather to talk SSL/TLS or plaintext.

The HTTP test makes a GET request to the target and once the headers come in, it terminates the connection without consuming the full body. This is by design so as to not consume too much bandwidth.

#### mtr/traceroute

mtr test is a wrapper around the mtr command.

API endpoint: /mtr/
Method: POST
Payload: Json object

example :-

	{
		"Target": "example.com"
	}

`Target` : The hostname/ip we want to trace to.

## Create new test types

coming soon...