# Client Group Endpoints

Blockasaurus can identify which client group a DNS query belongs to using
hostnames, URL paths, or EDNS options — similar to how NextDNS assigns
per-profile hostnames. This lets you give each client group its own DNS
endpoint without relying solely on source IP matching.

## How It Works

Each client group gets a **slug** — a DNS-safe identifier derived from the
group name. For example, a group named "Kids Devices" gets the slug
`kids-devices`. This slug is used across all identification methods:

| Protocol | Identification method | Example |
|----------|----------------------|---------|
| Plain DNS (UDP/TCP) | EDNS CPE-ID (dnsmasq `add-cpe-id=`) | `add-cpe-id=kids-devices` |
| Plain DNS (UDP/TCP) | Source IP / CIDR | Existing behavior |
| DoH (HTTPS) | URL path | `https://dns.example.com/dns-query/kids-devices` |
| DoH (HTTPS) | Subdomain | `https://kids-devices.dns.example.com/dns-query` |
| DoT (TLS, port 853) | TLS SNI | `kids-devices.dns.example.com` |
| DoQ (QUIC, port 853) | TLS SNI | `kids-devices.dns.example.com` |

**Resolution priority** (first match wins):

1. DoH URL path parameter (`/dns-query/{slug}`)
2. Subdomain from HTTP Host header or TLS SNI
3. EDNS CPE-ID option
4. Source IP / reverse DNS name (existing behavior)

## Configuration

```yaml
clientGroupEndpoints:
  # Base domains for subdomain-based identification.
  # {slug}.{domain} in TLS SNI or HTTP Host extracts the slug.
  domains:
    - dns.example.com
    - blockasaurus.local

  # Enable EDNS CPE-ID extraction (option code 65074) for plain DNS.
  # Used by dnsmasq's add-cpe-id= directive.
  cpeId: true     # default: true
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientGroupEndpoints.domains` | list of strings | `[]` | Base domains for subdomain extraction |
| `clientGroupEndpoints.cpeId` | bool | `true` | Enable EDNS CPE-ID (option 65074) extraction |

## Setup Scenarios

### Scenario 1: Homelab — No TLS

The simplest setup. Clients are identified by source IP, EDNS CPE-ID, or
DoH URL path. No TLS certificates or custom domain required.

```yaml
clientGroupEndpoints:
  cpeId: true

ports:
  dns: 53
  http: 4000
```

**Client examples:**

- **Any device (DHCP):** Set DNS server to blockasaurus IP (e.g., 192.168.1.5).
  The device's source IP is matched against client groups.
- **Subnet of IoT devices:** Add CIDR `192.168.1.0/28` to a client group.
- **dnsmasq forwarder:** Forward to blockasaurus and identify with CPE-ID:
  ```
  server=192.168.1.5
  add-cpe-id=kids-devices
  ```
- **Browser/OS DoH:** Configure DoH URL:
  `http://192.168.1.5:4000/dns-query/kids-devices`

!!! note
    Android Private DNS and other DoT-only clients are not available without
    TLS. Use Scenario 2 or 3 for those.

---

### Scenario 2: Homelab with Self-Signed Wildcard Cert (.local)

Adds subdomain-based DoH and DoT. Blockasaurus resolves its own subdomains
via a CustomDNS wildcard entry.

```yaml
clientGroupEndpoints:
  domains:
    - blockasaurus.local
  cpeId: true

ports:
  dns: 53
  http: 4000
  https: 443
  tls: 853

certFile: /certs/tls.crt
keyFile: /certs/tls.key

customDNS:
  mapping:
    blockasaurus.local: 192.168.1.5
    "*.blockasaurus.local": 192.168.1.5
```

**Generate the self-signed wildcard cert:**

```bash
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -days 3650 -nodes \
  -keyout tls.key -out tls.crt \
  -subj '/CN=blockasaurus.local' \
  -addext 'subjectAltName=DNS:blockasaurus.local,DNS:*.blockasaurus.local'
```

**Client examples** (in addition to Scenario 1):

- **Browser DoH (subdomain):**
  `https://kids-devices.blockasaurus.local/dns-query`
- **DoT per-group:**
  `kids-devices.blockasaurus.local` port 853
- **Android Private DNS:**
  `kids-devices.blockasaurus.local` (device must trust the self-signed cert)

!!! important "How .local subdomain resolution works"
    mDNS (multicast DNS) does **not** support wildcard names. The subdomain
    `kids-devices.blockasaurus.local` is resolved by blockasaurus's own DNS
    via the `customDNS` wildcard entry — not by mDNS. This means clients
    must already be using blockasaurus as their DNS server (port 53) before
    subdomain-based DoH/DoT will work. This is a natural bootstrap
    dependency — your DHCP server hands out blockasaurus's IP as the DNS
    server first.

!!! tip "Distributing the self-signed cert"
    Each client must trust the self-signed certificate:

    - **Android:** Install as user CA in Settings → Security → Encryption
    - **iOS/macOS:** Install and trust via a configuration profile
    - **Linux:** Copy to `/usr/local/share/ca-certificates/` and run `update-ca-certificates`
    - **Windows:** Import into the Trusted Root Certification Authorities store

---

### Scenario 3: Custom Domain with Wildcard Cert

Full production setup with a real domain and automated wildcard cert from
Let's Encrypt. All identification methods are available.

```yaml
clientGroupEndpoints:
  domains:
    - dns.example.com
  cpeId: true

ports:
  dns: 53
  http: 4000
  https: 443
  tls: 853

certFile: /certs/tls.crt
keyFile: /certs/tls.key
```

**DNS records** (at your registrar or DNS provider):

```
dns.example.com.       A     203.0.113.5
*.dns.example.com.     A     203.0.113.5
```

**TLS cert** (Let's Encrypt wildcard via DNS-01 challenge):

```bash
certbot certonly --dns-cloudflare \
  -d 'dns.example.com' \
  -d '*.dns.example.com'
```

**Client examples** (all methods):

| Device / Software | Configuration | Method |
|---|---|---|
| Any device (DHCP) | DNS server: 203.0.113.5 | Source IP |
| dnsmasq forwarder | `server=203.0.113.5` / `add-cpe-id=kids-devices` | EDNS CPE-ID |
| Browser DoH (path) | `https://dns.example.com/dns-query/kids-devices` | URL path |
| Browser DoH (subdomain) | `https://kids-devices.dns.example.com/dns-query` | Subdomain |
| Android Private DNS | `kids-devices.dns.example.com` | DoT via SNI |
| iOS/macOS profile | `https://kids-devices.dns.example.com/dns-query` | DoH via subdomain |
| DoQ client | `quic://kids-devices.dns.example.com:853` | DoQ via SNI |
| Stubby (DoT) | Server: `kids-devices.dns.example.com`, port 853 | DoT via SNI |

This is the recommended setup for homelabs with a domain name. Let's Encrypt
wildcard certs are free and auto-renewable, and all clients work without
manual certificate trust.

## Slug Generation

Client group names are automatically converted to DNS-safe slugs:

| Group Name | Slug |
|------------|------|
| Kids Devices | `kids-devices` |
| IoT_Network | `iot-network` |
| Guest WiFi 2.4 | `guest-wifi-2-4` |

Rules: lowercase, spaces/underscores/dots become hyphens, non-alphanumeric
characters are stripped, max 63 characters (DNS label limit). Each slug must
be unique across all client groups.

The slug is shown alongside the group name in the web UI and is the value
used for DoH paths, subdomain hostnames, and CPE-ID strings.
