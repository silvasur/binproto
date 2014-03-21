package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/kch42/binproto"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	mode  = flag.String("mode", "", "Mode. Either 'client' or 'proxy'")
	raddr = flag.String("raddr", "", "Address to connect to.")
	laddr = flag.String("laddr", "[::1]:31337", "Address to listen to (for proxy mode).")
)

func dedent(s string) string {
	l := len(s)
	if l > 0 {
		s = s[1:]
	}
	return s
}

func displayIncoming(r io.Reader, prefix string) {
	indent := ""
	out := func(f string, args ...interface{}) {
		fmt.Printf("%s%s"+f+"\n", append([]interface{}{prefix, indent}, args...)...)
	}

	ur := binproto.NewSimpleUnitReader(r)

	for {
		ut, data, err := ur.ReadUnit()
		switch err {
		case nil:
		case io.EOF:
			return
		default:
			fmt.Fprintf(os.Stderr, "could not read next unit: %s\n", err)
			os.Exit(1)
		}

		switch ut {
		case binproto.UTNil:
			out("Nil")
		case binproto.UTRequest:
			out("Request %d", data.(uint16))
		case binproto.UTAnswer:
			out("Answer %d", data.(uint16))
		case binproto.UTEvent:
			out("Event %d", data.(uint16))
		case binproto.UTBin:
			out("Bin %s", strconv.Quote(string(data.([]byte))))
		case binproto.UTNumber:
			out("Num %d", data.(int64))
		case binproto.UTList:
			out("List")
			indent += " "
		case binproto.UTTextKVMap:
			out("TextKVMap")
			indent += " "
		case binproto.UTIdKVMap:
			out("IdKVMap")
			indent += " "
		case binproto.UTUKey:
			out("UKey %d", data.(byte))
		case binproto.UTBinStream:
			out("Binstream")
			dumper := hex.Dumper(os.Stdout)
			if _, err := io.Copy(dumper, data.(*binproto.BinstreamReader)); err != nil {
				dumper.Close()
				fmt.Fprintf(os.Stderr, "error while dumping binstream: %s\n", err)
				os.Exit(1)
			}
			if err := dumper.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error while dumping binstream: %s\n", err)
				os.Exit(1)
			}
		case binproto.UTTerm:
			out("Term")
			indent = dedent(indent)
		}
	}
}

func clientUsage() {
	fmt.Fprint(os.Stderr, `One of:
Nil
Request <num>
Answer <num>
Event <num>
Bin <go string>
Number <num>
List
TextKVMap
IdKVMap
UKey <num>
BinStream <file>
Term
`)
}

func getuint(parts []string, bits int) (uint64, bool) {
	if len(parts) != 2 {
		clientUsage()
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 0, 16)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse number: %s\n", err)
		clientUsage()
		return 0, false
	}
	return n, true
}

func client() int {
	conn, err := net.Dial("tcp", *raddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to '%s': %s\n", *raddr, err)
		os.Exit(1)
	}
	defer conn.Close()

	go func() {
		displayIncoming(conn, "")
		fmt.Fprintln(os.Stderr, "--- Connection closed by remote host")
		os.Exit(0)
	}()

	bufin := bufio.NewReader(os.Stdin)

readloop:
	for {
		line, err := bufin.ReadString('\n')
		switch err {
		case nil:
		case io.EOF:
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Could not read line: %s", err)
			return 1
		}
		line = line[:len(line)-1]

		parts := strings.SplitN(line, " ", 2)

		switch strings.ToLower(parts[0]) {
		case "nil":
			if err := binproto.SendNil(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "request":
			if n, ok := getuint(parts, 16); ok {
				if err := binproto.InitRequest(conn, uint16(n)); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
		case "answer":
			if n, ok := getuint(parts, 16); ok {
				if err := binproto.InitAnswer(conn, uint16(n)); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
		case "event":
			if n, ok := getuint(parts, 16); ok {
				if err := binproto.InitEvent(conn, uint16(n)); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
		case "bin":
			if len(parts) != 2 {
				clientUsage()
				continue readloop
			}
			s, err := strconv.Unquote(strings.TrimSpace(parts[1]))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not interpret string: %s\n", err)
				clientUsage()
				continue readloop
			}
			if err := binproto.SendBin(conn, []byte(s)); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "number":
			if len(parts) != 2 {
				clientUsage()
				continue readloop
			}
			n, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 0, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not parse number: %s\n", err)
				clientUsage()
				continue readloop
			}
			if err := binproto.SendNumber(conn, n); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "list":
			if err := binproto.InitList(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "textkvmap":
			if err := binproto.InitTextKVMap(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "idkvmap":
			if err := binproto.InitIdKVMap(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		case "ukey":
			if n, ok := getuint(parts, 8); ok {
				if err := binproto.SendUKey(conn, byte(n)); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
		case "binstream":
			if len(parts) != 2 {
				clientUsage()
				continue readloop
			}

			if func(fname string) bool {
				f, err := os.Open(fname)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not open '%s': %s\n", fname, err)
					return false
				}
				defer f.Close()

				bsw, err := binproto.InitBinStream(conn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not init binstream: %s\n", err)
					return true
				}
				defer bsw.Close()

				if _, err = io.Copy(bsw, f); err != nil {
					fmt.Fprintf(os.Stderr, "Could not copy file to binstream: %s\n", err)
					return true
				}
				return false
			}(strings.TrimSpace(parts[1])) {
				return 1
			}
		case "term":
			if err := binproto.SendTerm(conn); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		default:
			clientUsage()
		}
	}
	return 0
}

func proxy() {
	listener, err := net.Listen("tcp", *laddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not listen on '%s': %s\n", *laddr, err)
	}
	defer listener.Close()

	connL, err := listener.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Accept() failed: %s\n", err)
	}
	defer connL.Close()

	connR, err := net.Dial("tcp", *raddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to '%s', %s\n", *raddr, err)
	}
	defer connR.Close()

	l2r := io.TeeReader(connL, connR)
	r2l := io.TeeReader(connR, connL)

	exit := make(chan bool)

	dispInWrap := func(r io.Reader, prefix string, exit chan<- bool, onexit bool) {
		displayIncoming(r, prefix)
		exit <- onexit
	}

	go dispInWrap(l2r, "[l -> r] ", exit, false)
	go dispInWrap(r2l, "[r -> l] ", exit, true)

	if <-exit {
		fmt.Fprintln(os.Stderr, "--- Connection closed by remote host")
	}
}

func main() {
	flag.Parse()

	switch *mode {
	case "client":
		os.Exit(client())
	case "proxy":
		proxy()
	case "":
		flag.Usage()
		os.Exit(1)
	default:
		fmt.Fprint(os.Stderr, "Unknown mode: '%s'\n", *mode)
		os.Exit(1)
	}
}
