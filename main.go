package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
)

var (
	name         = flag.String("name", "GoZeroconfGo", "The name for the service.")
	hostname     = flag.String("host", "", "The hostname of the service (default: system hostname)")
	service      = flag.String("service", "_workstation._tcp", "Service type.")
	domain       = flag.String("domain", "local.", "Network domain.")
	ip           = flag.String("ip", "", "IP address the service should advertise")
	port         = flag.Int("port", 42424, "Service port.")
	waitTime     = flag.Int("wait", 0, "Duration in seconds to publish service (0 = forever).")
	excludeIfaces = flag.String("exclude-ifaces", "", "Comma-separated list of interface names to exclude")
	includeIfaces = flag.String("include-ifaces", "", "Comma-separated list of interface names to include")
)
// ifaceMatches checks if ifaceName starts with any prefix in list
func ifaceMatches(ifaceName string, prefixes []string) bool {
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" && strings.HasPrefix(ifaceName, prefix) {
			return true
		}
	}
	return false
}

// getLocalIPv4s returns all non-loopback IPv4 addresses based on include/exclude
func getLocalIPv4s(includePrefixes, excludePrefixes []string) []string {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		// skip down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// decide if this iface should be used
		if len(includePrefixes) > 0 {
			// only include interfaces matching include prefixes
			if !ifaceMatches(iface.Name, includePrefixes) {
				continue
			}
		} else {
			// exclude interfaces matching exclude prefixes
			if ifaceMatches(iface.Name, excludePrefixes) {
				continue
			}
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				ips = append(ips, ip4.String())
			}
		}
	}

	return ips
}
func main() {
	flag.Parse()

	// Parse excluded interfaces
	exclude := []string{}
	if *excludeIfaces != "" {
		exclude = strings.Split(*excludeIfaces, ",")
	}
	include := []string{}
	if *includeIfaces != "" {
		include = strings.Split(*includeIfaces, ",")
	}

	// Determine hostname
	host := *hostname
	if host == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
		host = h
	}

	// Determine IPs
	var ips []string
	if *ip != "" {
		ips = []string{*ip}
	} else {
		ips = getLocalIPv4s(include, exclude)
		if len(ips) == 0 {
			log.Fatal("Could not detect any local IPv4 addresses")
		}
	}

	server, err := zeroconf.RegisterProxy(
		*name,
		*service,
		*domain,
		*port,
		host,
		ips,
		[]string{"txtv=0", "lo=1", "la=2"},
		nil,
	)
	if err != nil {
		panic(err)
	}
	defer server.Shutdown()

	log.Println("Published service:")
	log.Println("- Name:", *name)
	log.Println("- Host:", host)
	log.Println("- Ex ifaces:", exclude)
	log.Println("- In ifaces:", include)
	log.Println("- IPs:", ips)
	log.Println("- Type:", *service)
	log.Println("- Domain:", *domain)
	log.Println("- Port:", *port)

	// Signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	var tc <-chan time.Time
	if *waitTime > 0 {
		tc = time.After(time.Second * time.Duration(*waitTime))
	}

	select {
	case <-sig:
	case <-tc:
	}

	log.Println("Shutting down.")
}
