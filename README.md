# S99Base

Shared Go building blocks for applications and libraries. Packages use only the
standard library and `golang.org/x`, and do not import one another.

| Package | Purpose |
| --- | --- |
| `release/ghreleases` | Reads GitHub releases and classifies their assets. |
| `release/mdlocale` | Splits Markdown into per-locale sections opened by `<!-- lang:xx -->` lines. |
| `release/version` | Compares release tags and trims one leading `v`. |
| `work/applife` | Shuts an application down in reverse order under one deadline. |
| `work/daemon` | Runs HTTP over a Unix socket and starts the daemon on demand. |
| `common/ttlcache` | Holds one value behind a TTL; concurrent misses share one fetch. |
| `common/pool` | Runs work through bounded goroutines, keeping input order. |

## Install

Requires Go 1.27. Detached daemon spawning requires Unix.

```sh
go get github.com/smegg99/s99base
```

## Checks

With Go, a C compiler for the race detector, and `just` installed:

```sh
just check
```

Runs formatting checks, `go vet`, and all tests with the race detector.

## License

[MIT](LICENSE).
