# FAQ / Troubleshooting

## General

### How do I check if StremThru is running?

Send a request to the health endpoint:

```sh
curl http://<stremthru-host>/v0/health
```

### How do I get debug information?

The debug endpoint provides detailed diagnostics including IP addresses, authentication status, and store token info:

```sh
curl -u username:password http://<stremthru-host>/v0/health/__debug__
```

::: info
The debug endpoint requires authentication.
:::

### How do I change the log level?

Set the [`LOG_LEVEL`](/configuration/#server) environment variable. Available levels: `trace`, `debug`, `info`, `warn`, `error`.

Set [`LOG_FORMAT`](/configuration/#server) to `json` for structured logging.

## Authentication

### I'm getting "Unauthorized" errors

- Verify your credentials are correctly set in [`STREMTHRU_AUTH`](/configuration/#authentication)
- Credentials format: `username:password` (multiple users separated by commas)
- For API requests, use the `X-StremThru-Authorization` header with Basic auth

### I can't access the Dashboard

The Dashboard is restricted to admin users. Make sure your username is listed in [`STREMTHRU_AUTH_ADMIN`](/configuration/#authentication).

## Store

### Store returns 502 (Bad Gateway) errors

This means StremThru cannot reach the debrid service. Check:

- The debrid service's status page for outages
- Your network connectivity
- Tunnel/proxy configuration if using [`STREMTHRU_STORE_TUNNEL`](/configuration/#tunnel-proxy)

### IP mismatch errors

Some debrid services require requests to come from the same IP used during authentication. Use the [debug endpoint](#how-do-i-get-debug-information) to check which IPs StremThru is using.

If your server IP differs from your home IP, configure a tunnel:

```sh
STREMTHRU_STORE_TUNNEL=realdebrid:forced
HTTP_PROXY=http://your-proxy:port
```

See [Tunnel / Proxy](/configuration/#tunnel-proxy) configuration for details.

### "Magnet limit exceeded"

Your debrid service's storage limit has been reached. Remove unused items from your debrid account.

## Streaming

### No streams appearing in Stremio

1. Verify your store credentials are correct — check the [debug endpoint](#how-do-i-get-debug-information)
2. Ensure the addon is properly installed in Stremio
3. Check that the content type (movie/series) is supported by your configured addon
4. For Torz: verify at least one indexer is configured
5. For Newz: verify Usenet servers and indexers are configured

### Streams are slow to appear

- **Indexer timeout** — increase [`STREMTHRU_STREMIO_TORZ_INDEXER_MAX_TIMEOUT`](/configuration/stremio-addons) (default: 10s)
- **Rate limiting** — check if your indexers are rate-limited in Dashboard > Settings > Rate Limit Configs
- **Too many indexers** — reduce the number of configured indexers

## Usenet

### NNTP connection or authentication errors

- Verify server hostname, port, and credentials in Dashboard > Usenet > Servers
- Enable **TLS** if your provider requires it (typically port `563`)
- Use **Test Connection** in the dashboard to diagnose

### "Article not found" errors

This means the requested content segments aren't available on your Usenet provider:

- Add additional providers with different retention
- Add a **backup** provider — these are used when the primary provider can't find an article
- Some content may have been DMCA'd from your provider

### Encrypted or corrupted archive errors

Some NZB content uses password-protected or corrupted RAR archives. This is a limitation of the source content, not StremThru.

## Wrap Addon

### "Failed to fetch upstream manifests"

The upstream Stremio addon is unreachable:

- Verify the upstream addon URL is correct and accessible
- Check if the upstream addon requires authentication
- The upstream addon's server may be temporarily down

### Template or extractor errors

- Verify your template syntax — check [Stream Filter](/guides/stream-filter) for expression syntax
- Ensure the extractor URL is accessible
- Check logs for specific parsing error messages
