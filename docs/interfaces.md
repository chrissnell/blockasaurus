# Interfaces

## REST API


??? abstract "OpenAPI specification"

    ```yaml
    --8<-- "api/openapi.yaml"
    ```

If http listener is enabled, Blockasaurus provides a REST API. You can download the [OpenAPI YAML](api/openapi.yaml) interface specification. 

You can also browse the interactive API documentation (RapiDoc) documentation [online](rapidoc.html).

### Common endpoints

| Method | Path                  | Purpose                                              |
| ------ | --------------------- | ---------------------------------------------------- |
| GET    | `/api/blocking/enable`  | Enable blocking globally.                          |
| GET    | `/api/blocking/disable` | Disable blocking globally (optional `duration`, `groups` query params). |
| GET    | `/api/blocking/status`  | Return current blocking status as JSON.            |
| POST   | `/api/lists/refresh`    | Refresh all allow/denylists.                       |
| POST   | `/api/cache/flush`      | Clear the entire DNS response cache.               |
| POST   | `/api/query`            | Run a DNS query through Blocky and return the result as JSON. |
| GET    | `/api/stats`            | Query totals and block rate as JSON, read from the Prometheus registry. |
| GET    | `/api/stats/overtime`   | Per-bucket time series — totals, blocked, per-client counts and mean upstream latency. |
| GET    | `/api/stats/overtime/clients` | Alias of `/api/stats/overtime`, kept for UI compatibility. |
| GET    | `/api/stats/overtime/latency` | Alias of `/api/stats/overtime`, kept for UI compatibility. |
| GET    | `/api/stats/query-types`| Counts per DNS query type (`A`, `AAAA`, ...). |
| GET    | `/api/stats/response-types` | Counts per response type (`CACHED`, `BLOCKED`, ...). |
| GET    | `/api/stats/top-domains`| Top permitted and top blocked domains. |
| GET    | `/api/stats/top-clients`| Top clients by total and by blocked queries. |

!!! example "Flush the DNS cache"

    ```sh
    curl -X POST http://<blocky-host>:<http-port>/api/cache/flush
    ```

    Returns HTTP `200` on success. Useful after editing `customDNS`
    or `hostsFile` entries that may already be cached.

!!! note "Statistics semantics"

    The `/api/stats/*` series come from Blockasaurus' own statistics collector
    (`pkg/statscollector`), which persists its buckets in the SQLite config store so the
    dashboard survives a restart. They are independent of `prometheus.enable`: the collector is
    fed by the metrics resolver whether or not the Prometheus endpoint is exposed.

    `/api/stats` itself is the exception — it reads the Prometheus registry directly and reports
    `total_queries`, `blocked_queries` and `block_rate` (percent). Those counters are only
    incremented when `prometheus.enable` is true, so with Prometheus off this endpoint reports
    zeroes while the `/api/stats/*` series keep working.

    Clients are identified by their resolved name (see
    [client name lookup](configuration.md#client-name-lookup)) and fall back to their IP when no
    name is available. Queries dropped by the
    [rate limiter](configuration.md#rate-limiting-per-client-ip) are always attributed to the
    client IP: the limiter runs before the client name lookup so that its bucket key stays the
    connection's source IP.

    `FILTERED` and `NOTFQDN` responses do not reach the metrics resolver — the `filtering` and
    `fqdnOnly` resolvers answer those queries above it in the chain — so they are absent from
    both these series and the Prometheus counters.

## CLI

Blockasaurus ships a small CLI alongside the server; the binary is `blockasaurus`. Blocking, list
refresh and ad-hoc queries are driven from the web UI or the REST API above — upstream's
corresponding subcommands (`blocking`, `query`, `lists`, `stats`) are not part of this fork.

- `./blockasaurus serve [--config /path/to/config.yml]` starts the DNS server (the default command)
- `./blockasaurus validate [--config /path/to/config.yml]` validates the configuration file
- `./blockasaurus user ...` manages web UI users (see `./blockasaurus user --help`)
- `./blockasaurus healthcheck` probes a running server, and is what the container HEALTHCHECK runs
- `./blockasaurus version` prints the version
- `./blockasaurus completion <shell>` writes a shell completion script

!!! tip

    To run this inside docker run `docker exec blockasaurus ./blockasaurus version`
