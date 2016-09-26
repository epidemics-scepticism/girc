package main

import (
	"fmt"
	"strings"
)

func strsplit(s, sep string) (string, string) {
	a := strings.SplitN(s, sep, 2)
	empty := ""
	switch len(a) {
	case 2:
		return a[0], a[1]
	case 1:
		return a[0], empty
	default:
		return empty, empty
	}
}

func (i *IrcTerm) handleInput(cmd string) error {
	var e error = nil
	if strings.HasPrefix(cmd, "/") {
		cmd, args := strsplit(cmd[1:], " ")
		if f, ok := i.cmds[cmd]; ok {
			e = f(args)
		} else {
			i.i.out <- fmt.Sprintf("Unknown command \"%s\" - try /help", cmd)
		}
	} else if len(cmd) > 0 {
		if f, ok := i.cmds["msg"]; ok {
			e = f(fmt.Sprintf("%s %s", i.i.lastRcpt, cmd))
		}
	}
	i.update()
	return e
}

func (i *IrcTerm) cmdQuote(args string) error {
	return i.i.sendLine(args)
}

func (i *IrcTerm) cmdMsg(args string) error {
	rcpt, msg := strsplit(args, " ")
	i.i.lastRcpt = rcpt
	if len(msg) > 0 {
		if strings.HasPrefix(rcpt, "#") {
			return i.i.sendLine(fmt.Sprintf("PRIVMSG %s :%s", rcpt, msg))
		} else {
			msgs, _, e := o.Send([]byte(rcpt), []byte(msg))
			if e != nil {
				return e
			}
			for _, msg := range msgs {
				i.i.sendLine(fmt.Sprintf("PRIVMSG %s :%s", rcpt, string(msg)))
			}
		}
	}
	return nil
}

func (i *IrcTerm) cmdOtrStart(args string) error {
	if len(args) > 0 {

	}
	return nil
}

func (i *IrcTerm) cmdJoin(args string) error {
	i.i.lastRcpt = args
	return i.i.sendLine(fmt.Sprintf("JOIN %s", args))
}

func (i *IrcTerm) cmdNick(args string) error {
	return i.i.sendLine(fmt.Sprintf("NICK %s", args))
}

func (i *IrcTerm) cmdPart(args string) error {
	ch, rsn := strsplit(args, " ")
	return i.i.sendLine(fmt.Sprintf("PART %s :%s", ch, rsn))
}

func (i *IrcTerm) cmdQuit(args string) error {
	return i.i.sendLine(fmt.Sprintf("QUIT :Leaving."))
}

func (i *IrcTerm) cmdCtcp(args string) error {
	rcpt, ctcp := strsplit(args, " ")
	return i.i.sendLine(fmt.Sprintf("PRIVMSG %s :%s", rcpt, ctcp))
}

func (i *IrcTerm) cmdHelp(args string) error {
	i.i.out <- fmt.Sprintf("/join <channel>[,channel2,...,channeln]")
	i.i.out <- fmt.Sprintf("/msg <rcpt> [msg]")
	i.i.out <- fmt.Sprintf("/part <channel>")
	i.i.out <- fmt.Sprintf("/quit")
	return nil
}
