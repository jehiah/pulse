package net

import (
	"time"
)

// DialTimer connects to the address on the named network.
//
// See func Dial for a description of the network and address
// parameters.
//
// DialTimer is a clone of Dial, but instrumented to ptovide some extra timing information
func (d *Dialer) DialTimer(network, address string) (Conn, time.Duration, time.Duration, error) {
	finalDeadline := d.deadline(time.Now())
	start := time.Now()
	addrs, err := resolveAddrList("dial", network, address, finalDeadline)
	dnstime := time.Since(start)
	if err != nil {
		return nil, dnstime, time.Duration(0), &OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
	}
	start = time.Now()
	ctx := &dialContext{
		Dialer:        *d,
		network:       network,
		address:       address,
		finalDeadline: finalDeadline,
	}

	var primaries, fallbacks addrList
	if d.DualStack && network == "tcp" {
		primaries, fallbacks = addrs.partition(isIPv4)
	} else {
		primaries = addrs
	}

	var c Conn
	if len(fallbacks) == 0 {
		// dialParallel can accept an empty fallbacks list,
		// but this shortcut avoids the goroutine/channel overhead.
		c, err = dialSerial(ctx, primaries, nil)
	} else {
		c, err = dialParallel(ctx, primaries, fallbacks)
	}

	if d.KeepAlive > 0 && err == nil {
		if tc, ok := c.(*TCPConn); ok {
			setKeepAlive(tc.fd, true)
			setKeepAlivePeriod(tc.fd, d.KeepAlive)
			testHookSetKeepAlive()
		}
	}
	return c, dnstime, time.Since(start), err
}
