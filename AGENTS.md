# Agent guidelines

Instructions for AI coding agents (Claude Code, Copilot, Cursor, etc.) working in this repo.

## Project overview

`redis-benchmark-go` is a Go utility (22 stars) that replicates and extends the functionality of the official `redis-benchmark` tool. It supports load-testing Redis with configurable clients, keyspace, data sizes, RPS limits, RESP2/RESP3, OSS cluster mode, MULTI/EXEC pipelines, WAIT-for-replicas, and client-side caching (CSC) via the rueidis client. The binary accepts arbitrary Redis commands with `__key__` and `__data__` placeholders and emits per-second throughput and latency percentile (p50/p95/p99) summaries. Pre-built binaries for Linux, macOS, and Windows are published on each release; Go users can also install from source.

## Local setup

```bash
# Clone the repo
git clone git@github.com:redis-performance/redis-benchmark-go.git
cd redis-benchmark-go

# Fetch dependencies
GO111MODULE=on go get -t -v ./...

# Build the binary
make build
```

Go 1.20 or later is required. The binary will be placed in the repository root as `redis-benchmark-go`.

## Branch naming

Same as human contributors: `<type>/<short-description>` (e.g. `fix/off-by-one-in-pipeline`).

## Coding standards

- Match the style already in the file you are editing.
- Prefer clear, minimal changes over large refactors unless explicitly asked.
- Do not add comments that describe *what* the code does — only add comments when the *why* is non-obvious.
- Do not introduce new dependencies without checking with the maintainer.
- Run `make checkfmt` before committing; CI enforces `gofmt` formatting.

## Running tests

The test suite requires a Redis instance reachable at `localhost:6379`. Override the address with the `REDIS_TEST_HOST` environment variable (format `host:port`). Use `REDIS_TEST_PASSWORD` if authentication is needed.

```bash
make test
```

`make test` fetches dependencies, builds a coverage-instrumented binary, runs `go test` with `GOCOVERDIR`, and prints a coverage summary. This is identical to what the CI workflow (`.github/workflows/build.yml`) runs.

Always run `make test` before declaring a task complete.

## How to submit changes

1. Create a branch: `git checkout -b <type>/<description>`.
2. Commit with a clear message focused on *why*, not *what*.
3. Open a pull request against `master`.
4. Do **not** push directly to `master`.

## What to avoid

- Do not reformat files unrelated to your change.
- Do not remove error handling or tests.
- Do not commit secrets, credentials, or large binary files.
- Do not amend published commits.
