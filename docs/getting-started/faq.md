# FAQ

## StremThru

### What is StremThru?

Companion for Stremio. It contains a bunch of Addons, Endpoints, Integrations, Tools etc. All of these compliments your Stremio usage.

### Do I need to self-host it?

No, you do not need to host StremThru yourself. You can use one of the public instances:

- [stremthru.13377001.xyz](https://stremthru.13377001.xyz)
- [stremthru.elfhosted.com](https://stremthru.elfhosted.com)
- [stremthrufortheweebs.midnightignite.me](https://stremthrufortheweebs.midnightignite.me)

### Can I self-host it?

Yes, absolutely. In fact, some of the features are exclusive to self-hosted instances, e.g. Content Proxy, Watched Library Sync etc.

### How do I check if StremThru is running?

Send a request to the health endpoint:

```sh
curl ${STREMTHRU_BASE_URL}/v0/health
```

## Features

### How to disable all features?

Set the [`STREMTHRU_FEATURE`](/configuration/features) environment variable to exclude everything:

```sh
STREMTHRU_FEATURE=-dmm_hashlist,-imdb_title,-stremio_list,-stremio_sidekick,-stremio_store,-stremio_torz,-stremio_wrap
```

## Authentication

### How to access the Dashboard?

Go to `${STREMTHRU_BASE_URL}/dash`.

The Dashboard is restricted to admin users.

### How to set the admin users?

Admin users are listed in [`STREMTHRU_AUTH_ADMIN`](/configuration/#stremthru-auth-admin) config. If it's missing, all the users from [`STREMTHRU_AUTH`](/configuration/#stremthru-auth) config are treated as admin.

## StremThru Store

### What exactly the "Enable WebDL" toggle does?

If you download files from web hosters (e.g. Mega, Zippyshare etc.) to your debrid service, enabling this toggle will include those _Web Downloads_ in the _Stremio Catalog_.
