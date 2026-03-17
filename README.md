<p align="center">
  <img src="assets/blockasaurus-title.svg" alt="Blockasaurus" width="300">
</p>

Blockasaurus is a blocking DNS server with a modern web UI for blocklist and user management. It aims to deliver Pi-Hole-style functionality with first-class support for non-Pi platforms, including Kubernetes. Based on [Blocky](https://github.com/0xERR0R/blocky) by Dmitri Herzog.

> **Important**: Blockasaurus is an independent fork. Please **do not** send support requests, bug reports, or feature requests for Blockasaurus to the Blocky project. If you have issues with Blockasaurus, please file them [here](https://github.com/chrissnell/blockasaurus/issues).

## Features

- **Modern Web UI** - Manage blocklists, users, and settings from your browser

- **Blocking** - Block DNS queries with external lists (ad-block, malware) and allowlisting

  - Allow/denylists per client group (kids, smart home devices, etc.)
  - Periodical reload of external allow/denylists
  - Regex support
  - Blocking of request domain, response CNAME (deep CNAME inspection) and response IP addresses (against IP lists)

- **Advanced DNS configuration** - not just an ad-blocker

  - Custom DNS resolution for certain domain names
  - Conditional forwarding to external DNS server
  - Upstream resolvers can be defined per client group

- **Performance** - Improves speed and performance in your network

  - Customizable caching of DNS answers for queries
  - Prefetching and caching of often used queries
  - Multiple external resolvers simultaneously
  - Low memory footprint

- **Various Protocols** - Supports modern DNS protocols

  - DNS over UDP and TCP
  - DNS over HTTPS (DoH)
  - DNS over TLS (DoT)

- **Security and Privacy** - Secure communication

  - DNSSEC, eDNS, and other modern DNS extensions
  - DNSSEC validation of upstream resolvers
  - Freely configurable blocking lists - no hidden filtering
  - DoH endpoint
  - Random upstream resolver selection for privacy
  - Blockasaurus does **NOT** collect any user data, telemetry, or statistics

- **Integration**

  - [Prometheus](https://prometheus.io/) metrics
  - [Grafana](https://grafana.com/) dashboards
  - Query logging to CSV or MySQL/MariaDB/PostgreSQL/Timescale
  - REST API
  - CLI tool

- **Simple installation** - single binary, single YAML config

  - Docker image with multi-arch support
  - Helm chart for Kubernetes deployment
  - Runs on x86-64 and ARM (Raspberry Pi)
