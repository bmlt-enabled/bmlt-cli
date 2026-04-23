# bmlt-cli

[![Test](https://github.com/bmlt-enabled/bmlt-cli/actions/workflows/test.yml/badge.svg)](https://github.com/bmlt-enabled/bmlt-cli/actions/workflows/test.yml)
[![Release](https://github.com/bmlt-enabled/bmlt-cli/actions/workflows/release.yml/badge.svg)](https://github.com/bmlt-enabled/bmlt-cli/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/bmlt-enabled/bmlt-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/bmlt-enabled/bmlt-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/bmlt-enabled/bmlt-cli.svg)](https://pkg.go.dev/github.com/bmlt-enabled/bmlt-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A small, dependency-free command-line client for the [BMLT](https://bmlt.app/) Semantic API. Query any Basic Meeting List Tool root server from your shell — find Narcotics Anonymous meetings, dump service bodies, export NAWS CSVs, inspect format definitions.

Single static Go binary, stdlib only.

## Install

### Pre-built binary (recommended)

Download the archive for your platform from the [latest release](https://github.com/bmlt-enabled/bmlt-cli/releases/latest), extract, and place `bmlt` on your `PATH`:

```sh
tar -xzf bmlt-cli_*_macos_arm64.tar.gz
sudo mv bmlt /usr/local/bin/
bmlt --version
```

### Via Go

```sh
go install github.com/bmlt-enabled/bmlt-cli@latest
```

### From source

```sh
git clone https://github.com/bmlt-enabled/bmlt-cli
cd bmlt-cli
make build         # → ./bmlt
make install       # → $GOBIN/bmlt
```

## Quick start

```sh
# Browse known root servers (cached for 24h)
bmlt servers
bmlt servers ohio

# Talk to a server by URL or by fuzzy name
bmlt -s https://bmlt.sezf.org/main_server/ info
bmlt -s "Ohio Region" info

# Find virtual Monday meetings
bmlt -s "Ohio Region" search --weekdays mon --venue-types virtual

# Find meetings within 5 km of a coordinate, only the fields you need
bmlt -s "Ohio Region" search \
     --near 39.96,-83.00 --radius-km 5 \
     --fields meeting_name,start_time,weekday_tinyint,location_text

# Print the URL without fetching (great for sharing)
bmlt -s "Ohio Region" --url search --weekdays fri --formats 17
```

Set `BMLT_SERVER` to skip `-s` on every call:

```sh
export BMLT_SERVER="Ohio Region"
bmlt search --weekdays sat
```

## Commands

| Command   | Endpoint           | Notes |
|-----------|--------------------|-------|
| `servers` | (aggregator list)  | Filterable list of every known BMLT root |
| `info`    | `GetServerInfo`    | Version, features, languages, aggregator status |
| `coverage`| `GetCoverageArea`  | Bounding box of all meetings |
| `keys`    | `GetFieldKeys`     | Available meeting field keys |
| `values`  | `GetFieldValues`   | Distinct values for one field (`-k`) |
| `formats` | `GetFormats`       | Format catalog (Open, Closed, Speaker, …) |
| `bodies`  | `GetServiceBodies` | Service body tree (ZF → RS → AS) |
| `search`  | `GetSearchResults` | The workhorse — see flags below |
| `changes` | `GetChanges`       | Meeting change history |
| `naws`    | `GetNAWSDump`      | CSV export for a service body |

Run `bmlt <command> -h` for command-specific flags.

## Search flags

```
--weekdays      mon,wed,fri  or  2,4,6  (1=Sun, prefix - to exclude)
--venue-types   in-person,virtual,hybrid  or  1,2,3
--formats       Format IDs (e.g. 17,29 — server-specific; check `bmlt formats`)
--formats-op    AND (default) | OR
--services      Service body IDs (negative excludes)
--recursive     When filtering by service body, include child bodies
--q             Full-text search (name, location, notes)
--near LAT,LNG  Geographic center
--radius-km N   Radius in km (negative N = "find N nearest meetings")
--radius N      Radius in miles
--starts-after  HH:MM
--starts-before HH:MM
--ends-before   HH:MM
--min-duration  HH:MM
--max-duration  HH:MM
--fields        Comma list — restrict response to these field keys (huge perf win)
--sort          Multi-key sort (e.g. weekday_tinyint,start_time)
--sort-by       Predefined: weekday | time | town | state | weekday_state
--page-size N
--page N
--ids           Specific meeting IDs (negative excludes)
--published     all | unpublished
--lang          Language code for format names
--root-ids      [aggregator only] limit to specific underlying roots
--field-key K --field-value V
                Search any field returned by `bmlt keys`
```

## Output formats

- Default — human-readable, grouped by weekday for `search`, tree for `bodies`, table elsewhere.
- `--json` — pretty-printed JSON of the raw response.
- `--tsml` — TSML-shaped JSON (search only).
- `--url` — print the constructed URL and exit.

## Server resolution

The `-s/--server` flag accepts either a URL or a fuzzy name:

- `-s https://bmlt.sezf.org/main_server/` — used as-is.
- `-s "Ohio Region"` — looked up against [`serverList.json`](https://github.com/bmlt-enabled/aggregator/blob/main/serverList.json) from the bmlt-enabled/aggregator repo. Cached locally for 24 hours; force a refresh with `bmlt servers --refresh`.

The aggregator server (`https://aggregator.bmltenabled.org/main_server/`) runs in **aggregator mode** — it federates every entry in the canonical list. Reach for it only when you need a true cross-server query; per-server roots are authoritative for their own region. See `--root-ids` on `search`.

## Gotchas

- Empty `[]` from BMLT is not always an error — invalid params are often silently ignored. If a result looks wrong, dump the URL with `--url` and inspect.
- Format IDs are **server-specific** — ID 17 on one root is not the same format on another.
- Weekdays are 1-indexed starting Sunday: Monday is **2**.
- `naws` requires `--service-body` (`-b`) and is CSV-only.

## Development

```sh
make help        # list targets
make test        # go test -race ./...
make cover       # write coverage.out + coverage.html
make lint        # gofmt + go vet
make snapshot    # local goreleaser build (no publish)
```

CI runs on every push to `main` and every pull request:

- `go vet`, `gofmt`, `staticcheck`
- `go test -race` on Go 1.24/1.25 × Linux/macOS
- Coverage uploaded to Codecov

Releases are cut by pushing a tag matching `v*`:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser then builds binaries for Linux, macOS, Windows, FreeBSD across amd64/arm64/arm/386 and publishes them to GitHub Releases with checksums and an auto-generated changelog.

## See also

- [BMLT Semantic API docs](https://github.com/bmlt-enabled/bmlt-semantic-api-documentation)
- [semantic.bmlt.app](https://semantic.bmlt.app) — interactive sandbox for building and testing URLs
- Language client libraries under [github.com/bmlt-enabled](https://github.com/bmlt-enabled): TypeScript, Python, Go, Ruby, PHP, Perl, Swift

## License

MIT — see [LICENSE](LICENSE).
