# Docker Deployment

Production deployment tips for running StremThru with Docker.

## Topics

This page will cover:

- Production Docker configuration
- Volume management for persistent data
- Restart policies
- Resource limits
- Docker Compose for multi-service setups
- Health checks
- Logging configuration

## Basic Production Setup

```yaml
services:
  stremthru:
    image: muniftanjim/stremthru
    ports:
      - 8080:8080
    env_file:
      - .env
    restart: unless-stopped
    volumes:
      - ./data:/app/data
```

::: info
Detailed documentation coming soon — contributions welcome.
:::
