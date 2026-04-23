# AGENTS.md

Guidance for AI coding agents (Claude Code, Cursor, Aider, Copilot Workspace, etc.) working in this repo. Humans should read [CONTRIBUTING.md](CONTRIBUTING.md) instead — this file repeats some context from there in a form optimized for agents.

## What this project is

`bmlt-cli` is a single-binary Go CLI for the [BMLT](https://bmlt.app/) Semantic API — an HTTP API for Narcotics Anonymous meeting data exposed by every BMLT root server. The CLI builds and fetches Semantic API URLs and renders responses for humans. There is no SDK, no auth, no database — just `net/http` against public endpoints.

The Semantic API URL shape is always:

```
{root}/client_interface/{json|jsonp|tsml|csv}/?switcher={endpoint}&...
```

If you need to understand BMLT itself before making a change, read these in order:

1. [`README.md`](README.md) — user-facing surface and examples
2. [BMLT Semantic API docs](https://github.com/bmlt-enabled/bmlt-semantic-api-documentation)
3. The live sandbox at <https://semantic.bmlt.app> for building/testing URLs interactively

## Hard constraints

These are not preferences — they are gates on whether a change can merge.

- **Stdlib only.** No new entries in `go.mod`. If a task seems to require a dependency, stop and ask the human.
- **`gofmt` clean.** CI fails on unformatted files.
- **`go vet` clean.** CI fails on vet warnings.
- **`go test -race ./...` passes.** New behavior needs a test.
- **Backward-compatible flags.** Once a flag has shipped in a release, do not rename or remove it without an explicit human instruction to bump the major version.
- **No comments narrating what code does.** Comments only for *why* — non-obvious constraints, BMLT quirks, workarounds. Identifiers should carry their own weight.
- **No premature abstractions.** Three similar lines is better than a clever helper. Don't introduce interfaces, generics, or factories without a concrete second caller.
- **No error handling for cases that can't happen.** Trust internal contracts and stdlib guarantees. Only validate at boundaries.

## Project layout

Flat, single-package (`package main`) at the repo root — Go convention for a single-binary CLI. **Do not move sources into `src/`.** If the codebase ever outgrows flat layout, the idiomatic split is `cmd/bmlt/main.go` + `internal/<pkg>/`, but that's not warranted at current size.

| File | Responsibility |
|------|----------------|
| `main.go`     | Subcommand dispatcher, version metadata, top-level usage |
| `client.go`   | HTTP client, URL builder, response decoding |
| `servers.go`  | Aggregator `serverList.json` fetch, on-disk cache (24h TTL), fuzzy lookup |
| `flags.go`    | Shared flag parsing (weekday names, venue type names, HH:MM, lat,lng) |
| `cmds.go`     | One `runFoo` handler per subcommand |
| `format.go`   | Human-readable output formatters per response shape |
| `*_test.go`   | Table-driven unit tests |

The handler/dispatch convention matters:

- `main.go` declares `subcommands` (a `map[string]bool`) and a `switch` statement. **Both must be updated** when adding or removing a subcommand — `extractCommand` uses the map to pull the subcommand token out from anywhere in the arg list, the switch dispatches it.
- Subcommand handlers in `cmds.go` are named `runFoo` and take `args []string`. They construct their own `flag.NewFlagSet`, attach common flags via `attachCommon`, parse, build a `url.Values`, and either print the URL (if `--url`) or call `Client.Fetch` and hand the body to a formatter.
- Output is human-readable by default. `--json` pretty-prints raw JSON. `--tsml` is search-only.

## Common tasks — recipes

### Add a new flag to `search`

1. In `runSearch` in `cmds.go`, add the flag variable and `fs.StringVar`/`BoolVar`/`IntVar` registration alongside the existing block.
2. Wire it into the `url.Values` further down using `addStr`/`addCSV`/`addInt` (or a custom branch if it needs derived params, like `--near` does for lat/long/radius).
3. If the flag value needs parsing (a name → ID mapping, time string, etc.), put the helper in `flags.go` with table-driven tests in `flags_test.go`.
4. Update the "Search flags" section in `README.md`.
5. Run `make test && make lint`.

### Add a new subcommand

1. Add the name (and any aliases) to the `subcommands` map in `main.go`.
2. Add a `case` in the dispatcher switch in `main()`.
3. Add a `runFoo(args []string)` handler in `cmds.go` following the existing pattern (FlagSet → attachCommon → parse → resolveClient → build params → fetch → format).
4. Add a row to the `## Commands` table in `README.md` and an entry in the `printUsage` body in `main.go`.
5. If the response shape is new, add a formatter in `format.go` with tests.

### Add a parser helper

Put it in `flags.go`. Pattern: accept a string, return parsed value(s) + error. Always table-driven test. See `parseWeekdays`, `parseVenueTypes`, `parseHHMM`, `parseLatLng` for the established style — they accept both human-friendly aliases (`mon`, `virtual`) and raw IDs (`2`, `2`), and respect a leading `-` for "exclude" semantics.

### Add or change output formatting

Formatters live in `format.go` and take an `io.Writer` plus a `[]byte` JSON body. They unmarshal into a typed struct (declared at the top of the function family, e.g. `Meeting`, `ServiceBody`, `Format`) and render via `text/tabwriter`. Tests use `bytes.Buffer` + `strings.Contains` assertions on the rendered text.

**Important BMLT quirk:** every numeric field in BMLT JSON arrives as a *string*. `id_bigint`, `weekday_tinyint`, `venue_type` — all strings. Don't try to unmarshal into `int`. Convert with `strconv.Atoi` on demand.

## BMLT-specific gotchas to keep in mind

These are easy ways to ship a wrong-looking change:

- **Weekdays are 1-indexed starting Sunday.** Monday is `2`, not `1`.
- **Venue types**: `1=in-person`, `2=virtual`, `3=hybrid`. There is no `0`.
- **Format IDs are server-specific.** ID 17 on one root is a different format on another. Don't hardcode IDs anywhere except tests.
- **Empty `[]` is not always an error.** BMLT often silently ignores invalid params and returns an empty array. If a test of "this filter excludes everything" passes against a real server with `[]`, that may mean the param wasn't recognized.
- **Format × endpoint matrix matters.** `GetNAWSDump` only works with `csv`. `GetSearchResults` works with `json`/`jsonp`/`tsml`. Mismatch → HTTP 422. The `naws` handler explicitly switches `Client.Format = "csv"`.
- **Aggregator mode is special.** A single known root (`https://aggregator.bmltenabled.org/main_server/`) federates other servers. `GetSearchResults` against it requires at least one filter or returns `[]`. The `--root-ids` flag is only meaningful there.
- **The `data_field_key` parameter (exposed as `--fields`) is a major perf win.** When testing against a large real server, default to passing it.

## Testing strategy

- Pure helpers (parsers, normalizers, formatters): table-driven tests, aim for 100% coverage. Existing files set the bar.
- HTTP client: `httptest.NewServer` to assert the request URL and headers we send, and to test response handling without hitting a real BMLT server.
- Subcommand handlers (`runFoo`): not currently unit-tested because they shell out to `os.Args`/`os.Exit`/`os.Stdout`. If you refactor a handler to be testable (extracting the body into a function that takes `io.Writer` and returns an error), add tests.
- **Do not add tests that hit real BMLT servers.** They're flaky, slow, and rude to the operators. Use `httptest`.

Run with `make test` (race detector on) or `make cover` for an HTML report.

## CI and release flow

- **Push to `main` / open a PR** → `.github/workflows/test.yml` runs vet/gofmt/staticcheck/build/test on Linux+macOS × Go 1.24/1.25 with Codecov upload.
- **Push a `v*` tag** → `.github/workflows/release.yml` invokes GoReleaser v2 to build 14 archives across Linux/macOS/Windows/FreeBSD × amd64/arm64/arm/386 and publishes a GitHub Release.

Don't push tags or trigger releases without explicit human instruction. Don't edit workflows or `.goreleaser.yaml` for incidental work — those changes need a real reason.

## Commits and PRs

- One change per PR. Keep diffs small.
- [Conventional Commits](https://www.conventionalcommits.org) prefixes (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, `ci:`) — the release changelog groups by these.
- PR description should say *why*, not *what* (the diff shows what).

## What you should refuse / escalate

- Adding a runtime dependency to `go.mod`.
- Removing or renaming a flag, subcommand, or output field that has shipped in a tagged release.
- Adding network calls to anywhere outside `client.go` / `servers.go`.
- Embedding API keys, credentials, or per-user data in the repo.
- Producing a binary commit or modifying `dist/`.
- Force-pushing or rewriting history on `main`.

If a task seems to require any of the above, stop and ask the human first.
