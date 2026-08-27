# Cross-cutting nitpick taxonomy — redis-benchmark-go, real precedent only

Grounded in this repo's actual merged-PR titles/history (46 PRs, 2020–2026), its real (thin) review
comments, and its own `AGENTS.md`/`CONTRIBUTING.md`. This is a small, mostly single-maintainer Go
project — several categories below are evidenced by real, *self-authored* bugfix commits (the
maintainer fixing their own past mistake) rather than a reviewer catching something in review, since
that is what this repo's actual history mostly consists of. Where a category is genuinely thin or
absent, this file says so rather than inventing a citation.

1. **Release/build/binary-publishing plumbing is this repo's single most recurring, real source of
   bugs.** A large fraction of this project's entire merged-PR history is literally titled around
   fixing the release process: "Fixed prebuilt binaries release process" (#15), "Improved/Secured
   release process. Using GH links to access the prebuilt binaries" (#14), "Simplified release process
   gh actions" (#11), "updated release action to the lastest" (#25), "Fixed artifact upload GH action
   permission" (#22), "Fixed released process of gh binaries" (#7), "Using Go 1.16.x to build
   darwin/arm64" (#8), "Kicked off release process via github actions" (#6), "Improved release
   process" (#5). This is real, repeated precedent that this project's release/CI/cross-platform-build
   plumbing (`.github/workflows/publish.yml`, `docker-publish-master.yml`, `Dockerfile`,
   `docker-build.sh`) is fragile and has broken in production multiple times. Any PR touching these
   files deserves a careful, literal read of the YAML/script, not a skim — this is exactly the kind of
   surface that has bitten this repo before.

2. **Locking on the concurrent benchmark hot path was a real, fixed problem — don't reintroduce it.**
   PR#10's own title: "Removed all locking on stats updates." The benchmark's live code
   (`redis-bechmark-go.go`) runs many `benchmarkRoutine` goroutines concurrently (one or more per
   client), aggregating into `hdrhistogram` latency structures via a `datapointsChan` channel rather
   than shared mutexes — the one exception being `cscInvalidationMutex`, used only for the
   client-side-caching invalidation counters. A PR that adds a new per-command counter or shared piece
   of benchmark state should follow the channel/aggregation pattern already established, not introduce
   a new mutex on the hot path — that's precisely the kind of change PR#10 removed for a reason (stats
   updates happen once per command, at very high call rates under load).

3. **Test coverage is real, explicit written doctrine — but this repo's history doesn't show it being
   litigated in review.** `CONTRIBUTING.md`: "All new behaviour must be covered by tests... Coverage
   should not decrease." `AGENTS.md` repeats this for AI agents. Codecov is wired into `build.yml` and
   posts coverage automatically (`CODECOV_ORG_TOKEN`). No surveyed PR shows a reviewer citing a
   specific coverage number to block or question a merge, so don't claim it's enforced in practice
   beyond CI simply reporting it — but do apply the written rule and note when new logic in
   `redis-bechmark-go.go`, `commands.go`, `cluster_conn.go`, or `standalone_conn.go` ships without a
   corresponding addition to `redis-bechmark-go_test.go`.
   Tests require a live Redis reachable at `localhost:6379` (overridable via `REDIS_TEST_HOST`) — a
   PR that changes connection/auth/TLS logic should be checked for whether it's actually exercised by
   the existing integration-style tests or only manually described in the PR body.

4. **CLI flag additions should include a `README.md`/help-text update — this repo's real PRs almost
   always do.** Sampled feature PRs (RESP3 support #19, `-u` auth flag #33, `--tls` flags #39,
   multi-command `-cmd`/`-cmd-ratio` #21, `-nameserver` #24) each shipped alongside a README and/or
   `cli.go` usage-string update in the same PR. A PR that adds a new flag to `cli.go` without a
   matching README/usage update is missing something this repo's real precedent consistently includes.

5. **A new flag/feature interacting with an existing one is worth a concrete, run-it-yourself check —
   not just a code read.** The one substantive real review comment in this repo's history
   (filipecosta90 on PR#19) didn't just read the diff; it ran the actual binary with the new flag
   against a real Redis instance and cross-checked the *externally observable* result (`redis-cli
   client list` showing the negotiated RESP version) rather than trusting that the flag was plumbed
   through correctly. For a PR that changes how the tool talks to Redis (protocol version, TLS, auth,
   cluster vs standalone routing, client-side caching), the same standard applies: if you can construct
   the equivalent verification from the diff and PR description, say what you'd want checked and why,
   rather than only reading the code path in the abstract.

6. **Backward compatibility of existing CLI flags/output format has no evidenced maintainer precedent
   in this repo — say so rather than inventing one.** Unlike category 1 (release plumbing) or category
   2 (locking), no surveyed PR or review comment in this repo explicitly weighs a breaking vs.
   non-breaking CLI/output change. If a PR under review changes an existing flag's meaning, an existing
   output column, or the JSON-ish per-second summary format, be honest that this repo's own history
   doesn't give a citable precedent here, and reason about the tradeoff on its own merits.

7. **`AGENTS.md`'s explicit, written agent-specific rules should be checked directly** when the diff
   looks machine-generated: no new dependency added without maintainer sign-off, no comments describing
   *what* code does (only *why*, when non-obvious), no unrelated reformatting, `make checkfmt` clean
   (CI enforces `gofmt`).

## What this taxonomy is honestly thin or silent on

- **CodeQL/Copilot bot catches.** `.github/workflows/codeql-analysis.yml` exists, but no surveyed PR
  shows an alert, a fix responding to one, or a Copilot review-bot comment. Don't cite either as real
  precedent for this repo.
- **Deep, multi-round dialectic review.** The surveyed history has no example of it at all — not even
  the single richest example another redis-performance repo might have. The two most substantive real
  artifacts are a one-comment manual verification (PR#19) and a one-line clarifying question (PR#41).
  Don't manufacture a longer back-and-forth than that ever happened.
- **Maintainer issue triage.** Real, dated evidence points the other way: issues from external users
  have gone without any maintainer comment (#37, #13), and one was closed by its own reporter after
  getting no response (#17, "no reply close"). Frame automated triage comments honestly — as a
  best-effort first pass, not a promise that a human will necessarily follow up soon.
- **Buffer sizing / memory-safety nitpicks in the C sense.** Not applicable in the C/C++ way — this is
  Go, though the equivalent (goroutine/channel correctness on the hot path, see category 2) is real and
  should be applied instead.
