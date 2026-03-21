# Usenet Setup

This guide walks you through setting up StremThru's Usenet support end-to-end - from adding your Usenet provider and Newznab indexers to streaming content in Stremio.

## Prerequisites

Before you begin, make sure you have:

- **StremThru running** with Vault
- **Usenet Provider** - you'll need access to NNTP server
- **Newznab Indexer** - you'll need access to Newznab indexer

::: info
[`Vault`](/configuration/#vault) is required for Usenet functionality.
:::

## Step 1: Add Usenet Servers

Navigate to **Dashboard > Usenet > Servers** and click **Add Server**.

Fill in the server details:

| Field           | Description                                                        |
| --------------- | ------------------------------------------------------------------ |
| Name            | A label for this server (e.g. `My Provider`)                       |
| Host            | The NNTP server hostname (e.g. `news.provider.com`)                |
| Port            | The NNTP port (typically `119` for plain or `563` for TLS)         |
| TLS             | Enable for encrypted connections (recommended)                     |
| Username        | Your Usenet provider account username                              |
| Password        | Your Usenet provider account password                              |
| Priority        | Lower numbers are tried first when multiple servers are configured |
| Backup          | Mark as backup, used only when article missing on primary servers  |
| Max Connections | Maximum simultaneous NNTP connections allowed for this server      |

Click **Test Connection** to verify the credentials and connectivity, then click **Save**.

::: tip
Add your preferred provider with a low priority number (e.g. `0`). If you have a provider with block account, add it as **Backup**.
:::

## Step 2: Add Newznab Indexers

Navigate to **Dashboard > Usenet > Indexers** and click **Add Indexer**.

Fill in the indexer details:

| Field      | Description                                                             |
| ---------- | ----------------------------------------------------------------------- |
| Name       | A label for this indexer (e.g. `My Indexer`)                            |
| URL        | The base URL of the Newznab-compatible indexer                          |
| API Key    | Your indexer API key                                                    |
| Rate Limit | Optional - limit requests per time period to avoid hitting indexer caps |

Click **Save** to add the indexer.

::: tip
You can add multiple indexers. Results from all enabled indexers are aggregated when searching for content.
:::

## Step 3: Install the Newz Stremio Addon

Navigate to **Stremio Addons > Newz** in your StremThru instance. Configure the options as needed, then click the install button to add the addon to Stremio.

**Indexer** controls where NZBs are searched:

| Value     | Description                                                              |
| --------- | ------------------------------------------------------------------------ |
| StremThru | Combines results from the Newznab indexers configured in the dashboard - user:pass from `STREMTHRU_AUTH` must be specified as the API Key |
| Generic   | Uses a Newznab-compatible indexer directly - provide the URL and API key |
| Torbox    | Uses Torbox's built-in indexer - only usable with the Torbox store       |

**Store** controls how content is downloaded and streamed:

| Value     | Description                                                                           |
| --------- | ------------------------------------------------------------------------------------- |
| StremThru | Streams using your configured Usenet servers and any stores in `STREMTHRU_STORE_AUTH` |
| TorBox    | Uses TorBox for downloading and streaming                                             |

::: info
When using the **StremThru** indexer or store, Usenet server or Newznab indexer must be configured in dashboard. The **Torbox** indexer and store can work independently without those.
:::

## Verifying the Setup

- **Check pool status** - navigate to **Dashboard > Usenet > Config** to see the status of your configured Usenet servers and verify connections are healthy.
