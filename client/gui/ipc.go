package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// linkIPCAddr is a fixed loopback port used purely to hand a rookery://
// link from a second launch of the app (the user clicked a link while
// Rookery was already running) to the first instance — independent of the
// single-instance mutex, which only decides who gets to proceed.
const linkIPCAddr = "127.0.0.1:47812"

// startLinkListener runs for the process's lifetime, adding every
// subscription link a later launch forwards to it. Call once, in a
// goroutine, after App.ctx is set.
func startLinkListener(app *App) {
	ln, err := net.Listen("tcp", linkIPCAddr)
	if err != nil {
		// Another process already bound this port. Shouldn't happen
		// alongside the single-instance mutex, but failing open (no
		// deep-link forwarding, rather than crashing) is the right call.
		slog.Warn("ipc: link listener", "error", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleLinkConn(app, conn)
	}
}

func handleLinkConn(app *App, conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	app.HandleExternalLink(scanner.Text())
}

// forwardLinkToRunningInstance hands link off to an already-running
// instance's listener. Reports whether it succeeded — the caller should
// exit either way once a second instance is detected, but this distinguishes
// "delivered" from "nothing was listening" for logging purposes.
func forwardLinkToRunningInstance(link string) bool {
	conn, err := net.DialTimeout("tcp", linkIPCAddr, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	fmt.Fprintln(conn, link)
	return true
}
