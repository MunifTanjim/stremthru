# Reverse Proxy

Examples for running StremThru behind a reverse proxy.

## Topics

This page will cover:

- Nginx configuration
- Caddy configuration
- SSL/TLS setup
- WebSocket proxying
- Headers and buffering

## Nginx

```nginx
server {
    listen 80;
    server_name stremthru.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Caddy

```
stremthru.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

::: info
Detailed documentation coming soon — contributions welcome.
:::
