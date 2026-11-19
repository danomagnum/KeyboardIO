//go:build windows || (linux && !android)

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"time"

	"github.com/danomagnum/gologix"
)

type inputAssembly struct {
	Keys int32
	X    int32
	Y    int32
}

type inputEndpoint struct {
	gologix.IOProvider[inputAssembly, struct{}]
}

// The scanner's dummy output assembly has no effect on the desktop.
func (*inputEndpoint) IOWrite([]gologix.CIPItem) error { return nil }

func newRouter(endpoint *inputEndpoint) *gologix.PathRouter {
	router := &gologix.PathRouter{}
	router.Handle([]byte{1, 0}, endpoint)    // Backplane slot 0
	router.Handle([]byte{0x20, 4}, endpoint) // Direct assembly path
	router.Handle([]byte{0x34, 4}, endpoint) // Electronic key prefix
	return router
}

func main() {
	bind := flag.String("bind", "127.0.0.1", "local IPv4 address to bind")
	poll := flag.Duration("poll", 5*time.Millisecond, "desktop input polling interval")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "KeyboardIO publishes mapped desktop-wide key states and cursor position over EtherNet/IP.")
		flag.PrintDefaults()
	}
	flag.Parse()

	addr, err := netip.ParseAddr(*bind)
	if err != nil || !addr.Is4() || addr.IsUnspecified() {
		log.Fatal("-bind must be a specific local IPv4 address")
	}
	if *poll <= 0 {
		log.Fatal("-poll must be positive")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, addr.String(), *poll); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, bind string, poll time.Duration) error {
	desktop, err := openDesktop()
	if err != nil {
		return err
	}
	defer desktop.Close()
	first, err := desktop.sample()
	if err != nil {
		return err
	}
	endpoint := &inputEndpoint{}
	endpoint.SetInputData(first)
	server := gologix.NewServer(newRouter(endpoint))

	tcp, err := net.Listen("tcp4", net.JoinHostPort(bind, "44818"))
	if err != nil {
		return fmt.Errorf("open EtherNet/IP TCP listener: %w", err)
	}
	defer tcp.Close()
	udp, err := net.ListenPacket("udp4", net.JoinHostPort(bind, "2222"))
	if err != nil {
		return fmt.Errorf("open EtherNet/IP UDP listener: %w", err)
	}
	defer udp.Close()

	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Serve(tcp, udp) }()
	log.Printf("Desktop-wide input polling every %s; press Ctrl+C to stop", poll)
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-serverErr:
			return err
		case <-ticker.C:
			state, err := desktop.sample()
			if err != nil {
				return err
			}
			endpoint.SetInputData(state)
		}
	}
}
