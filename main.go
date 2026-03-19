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

func ipsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	m := make(map[string]struct{}, len(a))
	for _, ip := range a {
		m[ip] = struct{}{}
	}

	for _, ip := range b {
		if _, ok := m[ip]; !ok {
			return false
		}
	}

	return true
}

func waitForIPs(include, exclude []string) []string {
	for {
		ips := getLocalIPv4s(include, exclude)
		if len(ips) > 0 {
			log.Println("Detected IPs:", ips)
			return ips
		}
		log.Println("No IP found, waiting for network...")
		time.Sleep(10 * time.Second)
	}
}

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

	var ips []string
	
	if *ip != "" {
		ips = []string{*ip}
	} else {
		ips = waitForIPs(include, exclude)
	}
	
	startServer := func(ips []string) *zeroconf.Server {
		log.Println("Starting mDNS with IPs:", ips)
	
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
			log.Fatal(err)
		}
		return server
	}
	
	server := startServer(ips)
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
	
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sig:
			log.Println("Shutting down (signal)")
			return
		case <-tc:
        	log.Println("Shutting down (timeout)")
        	return
	
		case <-ticker.C:
			if *ip != "" {
				continue // static IP, skip monitoring
			}
	
			newIPs := getLocalIPv4s(include, exclude)
	
			// If no IP → wait again (network dropped)
			if len(newIPs) == 0 {
				log.Println("Lost network, waiting...")
				newIPs = waitForIPs(include, exclude)
			}
	
			// If IP changed → restart mDNS
			if !ipsEqual(ips, newIPs) {
				log.Println("IP change detected:", newIPs)
	
				server.Shutdown()
				server = startServer(newIPs)
				ips = newIPs
			}
		}
	}
	
	log.Println("Shutting down.")
}
