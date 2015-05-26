// Things borrowed from https://github.com/btcsuite/btcd/blob/master/addrmanager.go
// because "github.com/btcsuite/btcd" wouldn't import for some reason.

package main

import (
	"encoding/base32"
	"github.com/btcsuite/btcd/wire"
	"net"
	"strconv"
	"strings"
)

var onioncatrange = net.IPNet{IP: net.ParseIP("FD87:d87e:eb43::"),
	Mask: net.CIDRMask(48, 128)}

func Tor(na *wire.NetAddress) bool {
	// bitcoind encodes a .onion address as a 16 byte number by decoding the
	// address prior to the .onion (i.e. the key hash) base32 into a ten
	// byte number. it then stores the first 6 bytes of the address as
	// 0xfD, 0x87, 0xD8, 0x7e, 0xeb, 0x43
	// this is the same range used by onioncat, part of the
	// RFC4193 Unique local IPv6 range.
	// In summary the format is:
	// { magic 6 bytes, 10 bytes base32 decode of key hash }
	return onioncatrange.Contains(na.IP)
}

// ipString returns a string for the ip from the provided NetAddress. If the
// ip is in the range used for tor addresses then it will be transformed into
// the relavent .onion address.
func ipString(na *wire.NetAddress) string {
	if Tor(na) {
		// We know now that na.IP is long enogh.
		base32 := base32.StdEncoding.EncodeToString(na.IP[6:])
		return strings.ToLower(base32) + ".onion"
	} else {
		return na.IP.String()
	}
}

// NetAddressKey returns a string key in the form of ip:port for IPv4 addresses
// or [ip]:port for IPv6 addresses.
func NetAddressKey(na *wire.NetAddress) string {
	port := strconv.FormatUint(uint64(na.Port), 10)
	addr := net.JoinHostPort(ipString(na), port)
	return addr
}
