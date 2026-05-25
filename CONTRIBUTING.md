# Contributing

We treat this repo as "Open Source" within Redis: anyone who clears the bar below is welcome to contribute.

## Local setup

```bash
# Clone the repo
git clone git@github.com:redis-performance/redis-benchmark-go.git
cd redis-benchmark-go

# Download module dependencies
go mod download

# Build the binary
make build
```

Go 1.21 or later is required (per `go.mod`). CI runs the test matrix against 1.20.x and 1.21.x.

## Branch naming

```
<type>/<short-description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Example: `feat/add-rps-limit`

## Coding standards

- Keep changes focused; one logical change per PR.
- Follow the conventions already present in the codebase (formatting, naming, error handling).
- No dead code, no commented-out blocks.
- Run `make checkfmt` (wraps `gofmt`) before pushing; CI will reject unformatted code.

## Submitting changes

1. Fork or create a branch from `master`.
2. Make your changes with clear, atomic commits.
3. Open a pull request against `master` with a descriptive title and summary.
4. Address review comments promptly; force-push to the same branch to update.

## Testing

- All new behaviour must be covered by tests.
- Existing tests must pass: run the test suite locally before opening a PR.
- Coverage should not decrease.
- The test suite requires a Redis instance on `localhost:6379`. Override with `REDIS_TEST_HOST=host:port`; use `REDIS_TEST_PASSWORD` if authentication is needed.

```bash
# Run the full test suite (fetches deps, builds a coverage-instrumented binary, runs tests, reports coverage)
make test
```

`make test` is exactly what CI runs (see `.github/workflows/build.yml`).

## Review process

- At least one maintainer approval is required before merge.
- CI must be green.
- Maintainers may request changes or close PRs that do not meet the bar — this is normal and not personal.
