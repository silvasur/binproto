# binproto

binproto is a simple binary protocol written in Go. It was originally a part of a larger (now discontinued) project that I'll probably never publish.

This was pretty much the first code that I've written in Go, so it contains some unidiomatic and ugly stuff. For example, it uses no reflection, which can make reading and writing data quite tedious.

Another bad thing: Currently the sending code has no buffering and therefore sends a *lot* of small TCP packets where a single larger one would be better, adding quite a bit of overhead.

Still, it's BinStream data type is quite nice: By sending a BinStream you open a data stream inside of the stream (yo, dawg...), allowing you to send arbitrary data without too much overhead.

I'll publish this code, despite it's many quirks. Perhaps someone has a use for it?

## Installation

`go get github.com/silvasur/binproto`

## Documentation

Either install the package and use a local godoc server or use [godoc.org](http://godoc.org/github.com/silvasur/binproto)

## Protocol definition

The protocol assumes a server and a client. Clients send requests to the server, the server answers with an answer. The server can also send an event message.

The protocol sends units over the connection, a unit is one byte that determines the unit type and a payload that is different for each unit type.

Here are the unit types:

	Number | Name      | Payload
	-------+-----------+--------------------------------------------------
	 0     | Nil       | no Payload
	 1     | Request   | 2 byte request code + another unit
	 2     | Answer    | 2 byte response code + another unit
	 3     | Event     | 2 byte event code + another unit
	 4     | Bin       | 4 byte length + binary data of that length
	 5     | Number    | 8 byte int64
	 6     | List      | more units terminated by the Term unit
	 7     | TextKVMap | multiple pairs of Bin (with(!) type byte) + any
	       |           | type. Terminated by the Term unit
	 8     | IdKVMap   | payload are multiple pairs of UKey + any type.
	       |           | Terminated by the Term Unit
	 9     | UKey      | 1 byte
	10     | BinStream | multiple pairs  of 4 byte(signed) length + binary
	       |           | data of that length. Terminated with negative
	       |           | length (MSB set)
	11     | Term      | no Payload
	12     | Bool      | a single byte interpreted as bool
	       |           | (0 = false, true otherwise)
	13     | Byte      | a single byte

## binprotodebug

binprotodebug is a debugging utility for a binproto-based protocol. It allows you to play the role of a client `-mode client` or can function as a proxy `-mode proxy`. It displays the data in a human readable form.
 