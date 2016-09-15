package main

import (
	"flag"
	"strings"
)

func main() {
	server := flag.String("server", "irc.oftc.net:6697", "irc server")
	proxy := flag.String("proxy", "tor", "socks5 proxy")
	flag.Parse()
	i := NewIrc()
	srv := strings.SplitN(*server, ":", 2)
	i.Server(srv[0], srv[1])
	i.Proxy(*proxy)
	t, e := NewIrcTerm(i)
	if e != nil {
		i.Close()
		return
	}
	defer t.Stop()
	t.Start()
}
