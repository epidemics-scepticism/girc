package main

import (
	"flag"
	"fmt"
	"strings"
)

func main() {
	server := flag.String("server", "irc.oftc.net:6697", "irc server")
	proxy := flag.String("proxy", "tor", "socks5 proxy")
	sasl := flag.String("sasl", "", "sasl certificate and key prefix (foo.crt, foo.key)")
	nick := flag.String("nick", "", "nick")
	tls := flag.Bool("tls", true, "use tls")
	badtls := flag.Bool("badtls", false, "skip tls verify")
	flag.Parse()
	i := NewIrc()
	if len(*sasl) > 0 {
		i.UseSasl(*sasl)
	}
	if len(*nick) > 0 {
		i.Nick(*nick)
	}
	if *badtls {
		i.UseTlsInsecurely()
	}
	i.UseTls(*tls)
	srv := strings.SplitN(*server, ":", 2)
	i.Server(srv[0], srv[1])
	i.Proxy(*proxy)
	t, e := NewIrcTerm(i)
	if e != nil {
		i.Close()
		return
	}
	defer t.Stop()
	e = t.Start()
	if e != nil {
		fmt.Println(e)
	}
}
