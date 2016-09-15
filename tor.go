package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func torProxy() string {
	var host, port string = "127.0.0.1", "9050"
	if x := os.Getenv("TOR_SOCKS_HOST"); len(x) > 0 {
		host = x
	}
	if x := os.Getenv("TOR_SOCKS_PORT"); len(x) > 0 {
		port = x
	}
	return host + ":" + port
}

func torIsolateAuth() (string, string) {
	var seed [32]byte
	rand.Read(seed[:])
	user := sha256.Sum256(seed[:])
	pass := sha256.Sum256(user[:])
	return hex.EncodeToString(user[:]), hex.EncodeToString(pass[:])
}
