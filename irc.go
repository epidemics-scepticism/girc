package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

type ircOpts struct {
	tls, tlsInsecure bool
	sasl             bool
	saslCert         string
	showGuff         bool
	proxyAuth        bool
	proxy            string
	proxyUsername    string
	proxyPassword    string
	server, port     string
	nick             string
}

type Irc struct {
	in        chan string
	out       chan string
	msgs      chan *msg
	conn      net.Conn
	opts      *ircOpts
	lastRcpt  string
	timeStamp time.Time
}

func NewIrc() *Irc {
	return &Irc{
		in:   make(chan string, 64),
		out:  make(chan string, 64),
		msgs: make(chan *msg, 64),
		opts: &ircOpts{
			tls:         true,
			tlsInsecure: false,
			showGuff:    true,
			nick:        generateNick(),
		},
		timeStamp: time.Now().UTC(),
	}
}

func (i *Irc) Server(server, port string) {
	i.opts.server = server
	i.opts.port = port
}

func (i *Irc) Proxy(proxy string) {
	if proxy == "tor" {
		i.useTor()
	} else {
		i.opts.proxy = proxy
	}
}

func (i *Irc) useTor() {
	i.opts.proxy = torProxy()
	i.opts.proxyAuth = true
	i.opts.proxyUsername, i.opts.proxyPassword = torIsolateAuth()
}

func (i *Irc) UseTls(b bool) {
	i.opts.tls = b
}

func (i *Irc) UseTlsInsecurely() {
	i.opts.tls = true
	i.opts.tlsInsecure = true
}

func (i *Irc) UseSasl(cert string) {
	i.opts.sasl = true
	i.opts.saslCert = cert
}

func (i *Irc) Nick(nick string) {
	i.opts.nick = nick
}

func (i *Irc) Connect() error {
	infoMsg := fmt.Sprintf("Connecting to %s:%s over %s, tls: %v...",
		i.opts.server,
		i.opts.port,
		i.opts.proxy,
		i.opts.tls)
	i.infoMsg(infoMsg)
	var auth *proxy.Auth
	if i.opts.proxyAuth {
		auth = &proxy.Auth{User: i.opts.proxyUsername, Password: i.opts.proxyPassword}
	} else {
		auth = nil
	}
	d, e := proxy.SOCKS5("tcp", i.opts.proxy, auth, new(net.Dialer))
	if e != nil {
		return e
	}
	i.conn, e = d.Dial("tcp", i.opts.server+":"+i.opts.port)
	if e != nil {
		return e
	}
	if i.opts.tls {
		e = i.doTLS()
		if e != nil {
			i.conn.Close()
			return e
		}
	}
	if i.opts.sasl {
		i.sendLine("CAP REQ :sasl")
		i.sendLine("AUTHENTICATE EXTERNAL")
		i.sendLine("AUTHENTICATE +")
		i.sendLine("CAP END")
	}
	i.sendLine(fmt.Sprintf("USER %s * localhost :%s", i.opts.nick, i.opts.nick))
	i.sendLine(fmt.Sprintf("NICK %s", i.opts.nick))
	return nil
}

func (i *Irc) Close() error {
	for m := range i.out {
		fmt.Println(m)
	}
	close(i.in)
	close(i.out)
	close(i.msgs)
	return i.conn.Close()
}

func (i *Irc) sendLine(s string) error {
	_, e := i.conn.Write([]byte(s + "\r\n"))
	return e
}

func (i *Irc) infoMsg(s string) {
	l := []byte(fmt.Sprintf(":localhost INFO localhost :%s", s))
	i.msgs <- i.parse(l)
}

func (i *Irc) doTLS() error {
	cipherSuites := []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384, // no pfs!
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256, // no pfs!
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,    // no pfs!
	}
	cfg := &tls.Config{
		ServerName:         i.opts.server,
		CipherSuites:       cipherSuites,
		InsecureSkipVerify: i.opts.tlsInsecure,
	}
	if i.opts.sasl {
		sasl, e := tls.LoadX509KeyPair(i.opts.saslCert+".crt", i.opts.saslCert+".key")
		if e != nil {
			return e
		}
		cfg.Certificates = []tls.Certificate{sasl}
	}
	tconn := tls.Client(i.conn, cfg)
	if e := tconn.Handshake(); e != nil {
		return e
	}
	state := tconn.ConnectionState()
	i.conn = net.Conn(tconn)
	i.infoMsg(fmt.Sprintf("Cipher:\t%s", ciphersuite(state.CipherSuite)))
	for k, cert := range state.PeerCertificates {
		certMsg := fmt.Sprintf("\r\nCert Chain: %d\r\n\tSubject: %q\r\n\tIssuer:\r\n\t%q\r\n\tFingerprint: %q",
			k,
			cert.Subject.CommonName,
			cert.Issuer.CommonName,
			fingerprint(cert.Raw),
		)
		i.infoMsg(certMsg)
	}
	return nil
}

func (i *Irc) parse(line []byte) *msg {
	line = bytes.Trim(line, "\r\n")
	m := &msg{}
	m.enc = false
	delta := time.Now().UTC().Sub(i.timeStamp) / time.Second
	m.timestamp = []byte(fmt.Sprintf("%03d", delta % 1000))
	if bytes.HasPrefix(line, []byte{':'}) {
		line = line[1:]
		m.host, line = split(line, []byte{' '})
		m.nick, m.host = split(m.host, []byte{'!'})
		m.user, m.host = split(m.host, []byte{'@'})
	}
	line, m.content = split(line, []byte{' ', ':'})
	m.cmd, line = split(line, []byte{' '})
	m.rcpt, m.args = split(line, []byte{' '})
	return m
}

func fingerprint(data []byte) string {
	raw := sha256.Sum256(data)
	rawSplit := bytes.Split(raw[:], []byte{})
	for k := range rawSplit {
		rawSplit[k] = []byte(fmt.Sprintf("%02x", rawSplit[k]))
	}
	lfp := l.Encode(raw[:10])
	return fmt.Sprintf("[%s] (%s)", string(bytes.Join(rawSplit, []byte{':'})), lfp)
}

func ciphersuite(c uint16) string {
	switch c {
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
		return "ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return "ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:
		return "ECDHE_ECDSA_WITH_AES_256_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:
		return "ECDHE_RSA_WITH_AES_256_CBC_SHA"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
		return "ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:
		return "ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:
		return "ECDHE_ECDSA_WITH_AES_128_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:
		return "ECDHE_RSA_WITH_AES_128_CBC_SHA"
	case tls.TLS_RSA_WITH_AES_256_GCM_SHA384: // no pfs!
		return "RSA_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_RSA_WITH_AES_128_GCM_SHA256: // no pfs!
		return "RSA_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_RSA_WITH_AES_256_CBC_SHA: // no pfs!
		return "RSA_RSA_WITH_AES_256_CBC_SHA"
	default:
		return "UNKNOWN"
	}
}
