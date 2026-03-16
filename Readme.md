# gmdns

`gmdns` is a lightweight Go-based mDNS/ZeroConf service announcer for Android/Linux devices. It allows you to **announce your device’s IP on the LAN**, making it discoverable without knowing the IP.

## Features

- Announce device with **custom service name, type, and port**  
- Auto-detect **IPv4 addresses**  
- Auto hostname detection if not provided  
- Supports interface filtering:
  - `-exclude-ifaces`: skip interfaces by prefix (e.g., `rmnet_data`)  
  - `-include-ifaces`: only use interfaces by prefix (overrides exclude)  
- Fully compatible with Android (ARM64) and Linux  
- Optional timeout or run forever mode  
- Single binary, no dependencies except Go runtime  

## Installation

### Build for Linux
git clone https://github.com/yourusername/gmdns.git
cd gmdns
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o gmdns

### Build for Android (ARM64)
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o gmdns

## Usage

./gmdns [flags]

### Flags

| Flag               | Default          | Description |
|-------------------|----------------|------------|
| `-name`           | `GoZeroconfGo`  | Service name |
| `-host`           | system hostname | Hostname of the device |
| `-service`        | `_workstation._tcp` | Service type |
| `-domain`         | `local.`        | Network domain |
| `-ip`             | (auto)          | IP to advertise |
| `-port`           | `42424`         | Port to advertise |
| `-wait`           | `0`             | Duration in seconds (0 = forever) |
| `-exclude-ifaces` | (none)          | Comma-separated interface prefixes to skip |
| `-include-ifaces` | (none)          | Comma-separated interface prefixes to include (overrides exclude) |

### Examples

1. Run on all interfaces:
./gmdns

2. Exclude mobile data interfaces:
./gmdns -exclude-ifaces=rmnet_data

3. Only use Wi-Fi:
./gmdns -include-ifaces=wlan0

4. Custom service name and port:
./gmdns -name MyPhone -port 8080

## Notes

- IPv4 only  
- Include list overrides exclude list  
- Can be run headless or with systemd/cron for startup  

## License

MIT License. See [LICENSE](LICENSE).

## Acknowledgements

- [grandcat/zeroconf](https://github.com/grandcat/zeroconf) for mDNS/ZeroConf library.
