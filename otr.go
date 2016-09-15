package main

import (
	"crypto/rand"

	"github.com/twstrike/otr3"
)

type Otr struct {
	k *otr3.DSAPrivateKey
	c map[string]*otr3.Conversation
}

func newOtr() *Otr {
	o := &Otr{
		c: map[string]*otr3.Conversation{},
		k: &otr3.DSAPrivateKey{},
	}
	o.k.Generate(rand.Reader)
	return o
}

var o = newOtr()

func (o *Otr) GetConversation(rcpt string) *otr3.Conversation {
	if c, ok := o.c[rcpt]; ok {
		return c
	}
	return o.startConversation()
}

func (o *Otr) Recv(s, m []byte) ([]byte, [][]byte, bool, error) {
	c := o.GetConversation(string(s))
	rp, rr, e := c.Receive(otr3.ValidMessage(m))
	ise := c.IsEncrypted()
	p := []byte(rp)
	r := [][]byte{}
	for _, v := range rr {
		r = append(r, []byte(v))
	}
	return p, r, ise, e
}

func (o *Otr) Send(s, m []byte) ([][]byte, bool, error) {
	c := o.GetConversation(string(s))
	rr, e := c.Send(otr3.ValidMessage(m))
	ise := c.IsEncrypted()
	r := [][]byte{}
	for _, v := range rr {
		r = append(r, []byte(v))
	}
	return r, ise, e
}

func (o *Otr) startConversation() *otr3.Conversation {
	c := &otr3.Conversation{}
	c.Policies.AllowV2()
	c.Policies.AllowV3()
	c.SetOurKeys([]otr3.PrivateKey{o.k})
	c.SetFragmentSize(400)
	return c
}
