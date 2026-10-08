package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	port, dataPath, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "taskify:", err)
		os.Exit(2)
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	s, err := openStore(dataPath, time.Now().UTC())
	if err != nil {
		log.Error("cannot open data file", "err", err)
		os.Exit(1)
	}
	ln, err := net.Listen("tcp", listenAddr(port))
	if err != nil {
		log.Error("cannot listen", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Handler:           newHandler(s, port, log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	log.Info("listening", "url", "http://"+ln.Addr().String())
	if err := srv.Serve(ln); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// The host is fixed on purpose: with no login, Taskify must never be reachable from another computer (FR-006).
func listenAddr(port int) string { return fmt.Sprintf("127.0.0.1:%d", port) }
