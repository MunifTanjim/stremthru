# Stream Filter

## Expression Syntax

Expr language is used for filter expression. Check the language definition at:

- https://expr-lang.org/docs/language-definition

## Available Fields

Everything from parsed torrent title:

- https://github.com/MunifTanjim/go-ptt/blob/main/README.md

- **`Addon.Name`** (`string`)
- **`BitRate`** (`int`): bit per second
- **`Category`** (`string`): `movie`, `series`, `anime` etc.
- **`Episode`** (`int`): episode number (`-1` for movies)
- **`File.Name`** (`string`)
- **`File.Size`** (`string`): file size (e.g., `10 GB`)
- **`File.Idx`** (`int`): file index (0-based)
- **`Hash`** (`string`): torrent info hash
- **`IsPrivate`** (`bool`)
- **`Raw.Name`** (`string`)
- **`Raw.Description`** (`string`)
- **`Season`** (`int`): season number (`-1` for movies)
- **`Seeders`** (`int`): number of seeders
- **`Subtitles`** (`[]string`): language codes
- **`Store.Name`** (string): `alldebrid`, `debrider`, `debridlink` `easydebrid`, `offcloud`, `pikpak`, `premiumize`, `realdebrid`, `torbox`
- **`Store.Code`** (string): `AD`, `DR`, `DL`, `ED`, `OC`, `PP`, `PM`, `RD`, `TB`
- **`Store.IsCached`** (bool): Cached on debrid service
- **`Store.IsProxied`** (bool): Stream is proxied

## Examples

**Exclude 4K Resolution**:

```go
Resolution != "4k"
```

**Exclude Season Packs**:

```go
!Complete && len(Seasons) <= 1 && len(Episodes) <= 1
```

**Better or Equal to WEB Quality and 1080p Resolution**:

```go
Resolution >= "1080p" && Quality >= "WEB"
```

**Exclude Unwanted Languages**:

```go
len(Languages) == 0 || "sk" in Languages || "cs" in Languages || "en" in Languages
```
