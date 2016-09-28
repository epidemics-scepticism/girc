package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"./terminal"
)

type msg struct {
	timestamp, nick, user, host, cmd, rcpt, content, args []byte
	enc                                                   bool
}

func (m *msg) printTimestamp() []byte {
	ts := []byte{}
	ts = append(ts, '[')
	ts = append(ts, m.timestamp...)
	ts = append(ts, ']', ' ')
	return ts
}

func (m *msg) printFromTo() []byte {
	var c string
	if m.enc {
		c = "green"
	} else {
		if bytes.HasPrefix(m.rcpt, []byte{'#'}) {
			c = "yellow"
		} else {
			c = "red"
		}
	}
	var out []byte
	out = append(out, '[')
	out = append(out, colourise(m.nick, c)...)
	out = append(out, '@')
	out = append(out, colourise(m.rcpt, c)...)
	out = append(out, ']', ' ')
	return out
}

func (m *msg) printNUH() []byte {
	var out []byte
	out = append(out, '[')
	out = append(out, m.nick...)
	out = append(out, '!')
	out = append(out, m.user...)
	out = append(out, '@')
	out = append(out, m.host...)
	out = append(out, ']', ' ')
	return out
}

func (i *IrcTerm) handleMsg() {
	defer i.Stop()
	defer i.s.Done()
	for {
		select {
		case <-i.d:
			return
		case m, ok := <-i.i.msgs:
			if !ok {
				return
			}
			if f, ok := i.msgs[string(m.cmd)]; ok {
				f(m)
			} else {
				i.msgNotice(m)
			}
		}
	}
}

func (i *IrcTerm) msgPing(m *msg) {
	i.i.sendLine(fmt.Sprintf("PONG :%s", string(m.content)))
}

func (i *IrcTerm) msgPrivmsg(m *msg) {
	var out []byte
	if p, r, ise, e := o.Recv(m.nick, m.content); e != nil {
		infoMsg := fmt.Sprintf("OTR error with %s: %s", string(m.nick), e)
		i.i.infoMsg(infoMsg)
		return
	} else {
		m.content = p
		for _, v := range r {
			reply := fmt.Sprintf("PRIVMSG %s :%s", string(m.nick), string(v))
			i.i.sendLine(reply)
		}
		m.enc = ise
		if len(p) == 0 {
			return
		}
	}
	out = append(out, m.printTimestamp()...)
	out = append(out, m.printFromTo()...)
	out = append(out, m.content...)
	i.printLine(out)
}

func (i *IrcTerm) msgNick(m *msg) {
	if i.i.opts.showGuff {
		var out []byte
		out = append(out, m.printTimestamp()...)
		out = append(out, colourise(m.nick, "yellow")...)
		out = append(out, []byte(" is now known as ")...)
		out = append(out, colourise(m.content, "yellow")...)
		i.printLine(out)
	}
}

func (i *IrcTerm) msgJoin(m *msg) {
	if i.i.opts.showGuff {
		var out []byte
		out = append(out, m.printTimestamp()...)
		out = append(out, m.printNUH()...)
		out = append(out, []byte("has joined ")...)
		out = append(out, colourise(m.content, "white")...)
		i.printLine(out)
	}
}

func (i *IrcTerm) msgPart(m *msg) {
	if i.i.opts.showGuff {
		var out []byte
		out = append(out, m.printTimestamp()...)
		out = append(out, m.printNUH()...)
		out = append(out, []byte("has left ")...)
		out = append(out, colourise(m.rcpt, "white")...)
		if len(m.content) > 0 {
			out = append(out, []byte(" [")...)
			out = append(out, colourise(m.content, "white")...)
			out = append(out, []byte("]")...)
		}
		i.printLine(out)
	}
}

func (i *IrcTerm) msgQuit(m *msg) {
	if i.i.opts.showGuff {
		var out []byte
		out = append(out, m.printTimestamp()...)
		out = append(out, m.printNUH()...)
		out = append(out, []byte("has quit")...)
		if len(m.content) > 0 {
			out = append(out, []byte(" [")...)
			out = append(out, colourise(m.content, "white")...)
			out = append(out, []byte("]")...)
		}
		i.printLine(out)
	}
}

func (i *IrcTerm) msgGeneric(m *msg, c string) {
	var out []byte
	out = append(out, m.printTimestamp()...)
	out = append(out, m.printFromTo()...)
	out = append(out, colourise(m.cmd, c)...)
	out = append(out, []byte(" ")...)
	if len(m.args) > 0 {
		out = append(out, []byte("[")...)
		out = append(out, colourise(m.args, c)...)
		out = append(out, []byte("] ")...)
	}
	out = append(out, colourise(m.content, c)...)
	i.printLine(out)
}

func (i *IrcTerm) msgNotice(m *msg) {
	i.msgGeneric(m, "yellow")
}

func (i *IrcTerm) msgError(m *msg) {
	i.msgGeneric(m, "magenta")
}

func (i *IrcTerm) msgInfo(m *msg) {
	i.msgGeneric(m, "white")
}

var escape *terminal.EscapeCodes = &terminal.EscapeCodes{}

func colour(c string) []byte {
	switch c {
	case "black":
		return escape.Black
	case "red":
		return escape.Red
	case "green":
		return escape.Green
	case "yellow":
		return escape.Yellow
	case "blue":
		return escape.Blue
	case "magenta":
		return escape.Magenta
	case "cyan":
		return escape.Cyan
	case "white":
		return escape.White
	case "reset":
		return escape.Reset
	default:
		return []byte{}
	}
}

func colourise(data []byte, choice string) []byte {
	out := colour(choice)
	out = append(out, data...)
	out = append(out, colour("reset")...)
	return out
}

func split(s, d []byte) ([]byte, []byte) {
	arr := bytes.SplitN(s, d, 2)
	empty := []byte{}
	if len(arr) == 2 {
		return arr[0], arr[1]
	} else if len(arr) == 1 {
		return arr[0], empty
	} else {
		return empty, empty
	}
}

type IrcTerm struct {
	t    *terminal.Terminal
	o    *terminal.State
	i    *Irc
	p    string
	d    chan bool
	cmds map[string]func(args string) error
	msgs map[string]func(m *msg)
	s    sync.WaitGroup
}

type stdrw struct {
	Reader io.Reader
	Writer io.Writer
}

func (s stdrw) Read(p []byte) (int, error) {
	return s.Reader.Read(p)
}

func (s stdrw) Write(p []byte) (int, error) {
	return s.Writer.Write(p)
}

func NewIrcTerm(i *Irc) (*IrcTerm, error) {
	state, e := terminal.MakeRaw(0)
	if e != nil {
		return nil, e
	}
	rw := stdrw{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
	var p string = "> "
	t := &IrcTerm{
		t: terminal.NewTerminal(rw, p),
		o: state,
		i: i,
		p: p,
	}
	t.cmds = map[string]func(string) error{
		"msg":   t.cmdMsg,
		"join":  t.cmdJoin,
		"quit":  t.cmdQuit,
		"part":  t.cmdPart,
		"ctcp":  t.cmdCtcp,
		"help":  t.cmdHelp,
		"nick":  t.cmdNick,
		"quote": t.cmdQuote,
	}
	t.msgs = map[string]func(m *msg){
		"PING":    t.msgPing,
		"PRIVMSG": t.msgPrivmsg,
		"JOIN":    t.msgJoin,
		"PART":    t.msgPart,
		"QUIT":    t.msgQuit,
		"NOTICE":  t.msgNotice,
		"ERROR":   t.msgError,
		"INFO":    t.msgInfo,
		"NICK":    t.msgNick,
	}
	escape = t.t.Escape
	t.d = make(chan bool, 1)
	t.update()
	resiz := make(chan os.Signal)
	go func() {
		for _ = range resiz {
			t.update()
		}
	}()
	signal.Notify(resiz, syscall.SIGWINCH)
	return t, nil
}

func (i *IrcTerm) Stop() error {
	if e := terminal.Restore(1, i.o); e != nil {
		i.i.out <- fmt.Sprintf("ERROR: %v", e)
	}
	if i.d != nil {
		tmp := i.d
		i.d = nil
		close(tmp)
	}
	i.s.Wait()
	return i.i.Close()
}

func (i *IrcTerm) Start() error {
	if e := i.i.Connect(); e != nil {
		return e
	}
	go i.handleMsg()
	i.s.Add(1)
	go i.handleOutput()
	i.s.Add(1)
	go i.handleInfo()
	i.s.Add(1)
	defer i.s.Wait()
	defer close(i.d)
	for {
		s, e := i.t.ReadLine()
		if e != nil {
			i.i.out <- fmt.Sprintf("ERROR: %s", e)
			i.Stop()
			return e
		}
		i.handleInput(s)
	}
}

func (i *IrcTerm) handleInfo() {
	defer i.Stop()
	defer i.s.Done()
	for {
		select {
		case <-i.d:
			return
		case info := <-i.i.out:
			info = fmt.Sprintf("[INFO] %s", info)
			if e := i.printLine([]byte(info)); e != nil {
				i.i.out <- fmt.Sprintf("ERROR: %s", e)
				return
			}
		}
	}
}

func (i *IrcTerm) handleOutput() {
	defer i.Stop()
	defer i.s.Done()
	c := bufio.NewReader(i.i.conn)
	for {
		select {
		case <-i.d:
			return
		default:
			data, e := c.ReadBytes('\n')
			if e != nil {
				i.i.out <- fmt.Sprintf("ERROR: %s", e)
				return
			}
			bytes.TrimRight(data, "\r\n")
			i.i.msgs <- i.i.parse(data)
		}
	}
}

func (i *IrcTerm) printLine(data []byte) error {
	data = append([]byte("\r"), data...)
	data = append(data, '\r', '\n')
	_, e := i.t.Write(data)
	return e
}

func (i *IrcTerm) update() {
	i.t.SetPrompt(i.i.lastRcpt + i.p)
	w, h, e := terminal.GetSize(1)
	if e != nil {
		return
	} else {
		i.t.SetSize(w, h)
	}
}
