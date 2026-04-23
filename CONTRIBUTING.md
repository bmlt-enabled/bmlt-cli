# Contributing to bmlt-cli

Thanks for your interest in improving `bmlt-cli`. This document covers how to set up a dev environment, the conventions the project follows, and how to get a change merged.

## Code of Conduct

Be kind, assume good intent, and remember the audience: this tool serves the recovery community. Discriminatory, harassing, or hostile behavior in issues, PRs, or any other project space is not welcome.

## Ground rules

- **One change per PR.** Small, focused PRs get reviewed faster.
- **No new dependencies without discussion.** This project is intentionally stdlib-only — please open an issue before adding anything to `go.mod`.
- **Tests required for new behavior.** Bug fixes need a regression test; new flags need parser/wiring coverage.
- **Backward-compatible flags.** Once a flag ships in a release, renaming or removing it is a breaking change and requires a major version bump.

## Setup

You need [Go 1.24+](https://go.dev/dl/) and (optionally) [GoReleaser v2](https://goreleaser.com/install/) for cutting local snapshot builds.

```sh
git clone https://github.com/bmlt-enabled/bmlt-cli
cd bmlt-cli
make build           # → ./bmlt
./bmlt servers ohio  # smoke test
```

Common loops:

```sh
make test            # go test -race ./...
make cover           # coverage report (writes coverage.out + coverage.html)
make lint            # gofmt + go vet
make snapshot        # full goreleaser build for every platform (no publish)
```

## Project layout

Everything lives at the repo root — Go convention for a single-binary CLI.

| File | Responsibility |
|------|----------------|
| `main.go`     | Subcommand dispatcher, version metadata, top-level usage |
| `client.go`   | HTTP client, URL builder, response decoding |
| `servers.go`  | Aggregator `serverList.json` fetch, cache, fuzzy lookup |
| `flags.go`    | Shared flag parsing helpers (weekday/venue/HHMM/lat-lng) |
| `cmds.go`     | One handler per subcommand |
| `format.go`   | Human-readable output formatters |
| `*_test.go`   | Unit tests (table-driven) |

The CLI talks to the BMLT **Semantic API** over plain HTTP — no SDK, no auth. If you're new to BMLT, the [Semantic API documentation](https://github.com/bmlt-enabled/bmlt-semantic-api-documentation) and the live sandbox at <https://semantic.bmlt.app> are the best starting points.

## Coding conventions

- `gofmt` — CI rejects unformatted files. `make fmt` rewrites in place.
- `go vet` clean.
- Stick to the standard library. The whole point of this tool is "one binary, no surprises."
- **No comments narrating what code does.** Identifiers should carry their weight. Comments are for *why* — non-obvious constraints, BMLT quirks, workarounds.
- Don't pre-emptively add error handling for cases that can't happen, or fallbacks for hypothetical futures.
- Prefer table-driven tests (`map[string]string` or `[]struct{...}` cases). See existing `*_test.go` files for the style.

## Adding a new flag to `search`

1. Add the flag definition + variable in `runSearch` in `cmds.go`.
2. Add the corresponding `addStr` / `addCSV` / `addInt` (or custom) wiring further down in the same function.
3. If the value needs parsing (weekday names, time strings, etc.), add a helper to `flags.go` with a unit test.
4. Document the flag under "Search flags" in `README.md`.
5. Update the relevant section of [the BMLT skill](https://github.com/bmlt-enabled/bmlt-skill) if the parameter is new or under-documented.

## Adding a new subcommand

1. Register the name in the `subcommands` map and the `switch` in `main.go`.
2. Add a `runFoo(args []string)` handler in `cmds.go`.
3. Add a row to the `## Commands` table in `README.md` and an entry in `printUsage` in `main.go`.
4. Tests: at minimum, parser tests if there's any non-trivial flag handling, and a `httptest`-backed test if behavior is interesting.

## Pull request checklist

Before opening a PR:

- [ ] `make test` passes locally
- [ ] `make lint` clean (`gofmt`, `go vet`)
- [ ] New behavior has tests
- [ ] `README.md` updated if user-visible behavior changed
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org) — `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, `ci:`. The release changelog groups by these prefixes, so they matter.

CI must be green for a PR to merge. The `Test` workflow runs on Linux + macOS across Go 1.24/1.25; the `Lint` job runs `gofmt`, `go vet`, and `staticcheck`.

## Releases

Maintainers cut releases by tagging `main`:

```sh
git tag v0.2.0
git push origin v0.2.0
```

GoReleaser then builds binaries for every supported platform, generates a changelog from commit prefixes, and publishes the GitHub Release. Versions follow [SemVer](https://semver.org):

- `MAJOR` — breaking flag changes, removed subcommands, output format changes that scripts might depend on
- `MINOR` — new subcommands, new flags, new output (additive)
- `PATCH` — bug fixes, doc fixes, dependency-free internal cleanup

## Reporting bugs

Open a GitHub issue with:

1. The exact command you ran (with `--url` output if relevant — easiest way to share what URL got built).
2. The BMLT root server URL involved (so we can reproduce against the same data).
3. What you expected vs. what happened, with the actual output if it's not too long.

For server-side BMLT bugs (e.g. invalid responses from a specific root server), please file those upstream at [bmlt-enabled/bmlt-server](https://github.com/bmlt-enabled/bmlt-server) — `bmlt-cli` is just a client.

## Questions

Open a GitHub Discussion or drop into the BMLT community channels linked from <https://bmlt.app>. Thanks for contributing.
