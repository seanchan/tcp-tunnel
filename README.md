# TCP Tunnel

[![Release Build](https://github.com/seanchan/tcp-tunnel/actions/workflows/release.yml/badge.svg)](https://github.com/seanchan/tcp-tunnel/actions/workflows/release.yml)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/seanchan/tcp-tunnel)](https://github.com/seanchan/tcp-tunnel/releases)
[![GitHub stars](https://img.shields.io/github/stars/seanchan/tcp-tunnel)](https://github.com/seanchan/tcp-tunnel/stargazers)
[![GitHub license](https://img.shields.io/github/license/seanchan/tcp-tunnel)](https://github.com/seanchan/tcp-tunnel/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/seanchan/tcp-tunnel)](https://goreportcard.com/report/github.com/seanchan/tcp-tunnel)

A lightweight TCP tunneling tool written in Go, allowing you to expose local services to the internet securely.

## Features

- Simple and lightweight TCP tunneling
- Support for multiple concurrent tunnels
- Cross-platform compatibility (Windows, macOS, Linux)
- Configurable port ranges
- Easy to use command-line interface

## Installation

### From Release

Download the latest release from our [releases page](https://github.com/seanchan/tcp-tunnel/releases).

### From Source 

```bash
git clone https://github.com/seanchan/tcp-tunnel.git
cd tcp-tunnel
go build -o tcp-tunnel ./cmd/server/
```

## Usage

### Server

```bash
./tcp-tunnel-server --port 8088 --min-port 10000 --max-port 20000
```

Options:
- `--port`: Server control port (default: 8088)
- `--min-port`: Minimum port for public endpoints (default: 10000)
- `--max-port`: Maximum port for public endpoints (default: 20000)

### Client

```bash
./tcp-tunnel-client --server localhost:8088 --local localhost:80
```

Options:
- `--server`: Server address (default: localhost:8088)
- `--local`: Local service address to tunnel (default: localhost:80)

## Example

1. Start the server:
```bash
./tcp-tunnel-server --port 8088
```

2. Start the client to expose a local web server:
```bash
./tcp-tunnel-client --server your-server:8088 --local localhost:3000
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Thanks to all contributors who have helped with the project
- Inspired by various TCP tunneling solutions

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=seanchan/tcp-tunnel&type=Date)](https://star-history.com/#seanchan/tcp-tunnel)
