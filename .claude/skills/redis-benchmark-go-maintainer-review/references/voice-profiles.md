# Voice profiles — real redis-benchmark-go reviewers

Mined from actual GitHub history on `redis-performance/redis-benchmark-go`
(`gh pr list --state all --limit 300`, `gh api .../pulls/<n>/reviews`, `/comments`, and
`/issues/<n>/comments`, plus `gh issue list --state all`), covering the repo's full history
(2020–2026, 46 PRs, 7 issues at time of mining). Read this alongside `nitpick-taxonomy.md`.

**Be honest about what this repo's history actually is, up front:** this is a small, largely
single-maintainer project. There is no example in the surveyed history of a multi-round,
back-and-forth technical review, and no evidenced case of a reviewer tracing a bug through actual
code the way a project with a larger review culture might show. Do not manufacture a richer review
culture than exists. When a PR doesn't resemble any of the real patterns below, the honest thing is a
short, light-touch comment (or `skip_comment`), not an invented "maintainer voice."

## fcostaoliveira / filipecosta90 — Filipe Oliveira (same real person, two GitHub accounts: a Redis
work account and a personal account; by far the dominant author and approver)

Across the 46 surveyed PRs, one or the other of these two accounts authored the large majority, and
approved most of the rest (including PRs opened by the other account — i.e., largely self-review
across two identities). Most approvals are a bare `APPROVED` with an **empty body** — real examples:
PR#33, PR#35, PR#36, PR#38, PR#39, PR#40. That bare-approval pattern is the single most common thing
in this repo's history and should be treated as the default, not an exception needing an explanation.

The one real exception — and the sharpest, most concrete review artifact in this repo's whole
history — is **PR#19** (`ofekshenawa`'s RESP3 support). Rather than a bare approval, filipecosta90
wrote:

> "Nice work @ofekshenawa. approving. FYI did a quick smoke test to validate the tool is working as
> expected WRT protocol version:" — then pasted the actual command run
> (`./redis-benchmark-go -c 5 -p 6379 ... --resp 3 ...`), its real output, and the output of
> `redis-cli client list` on the target server showing `resp=3` in the connection list to independently
> confirm the negotiated protocol version, not just that the flag was accepted.

**What this means for the bot's voice**: this repo's one clearly evidenced review habit, when a
maintainer engages at all, is *independently running the feature and checking the actual observable
behavior it's supposed to produce* (not just reading the diff) — for a protocol/behavior-changing PR,
naming the specific command and expected observable side effect worth double-checking is grounded
in this exact precedent. For everything else, a bare "LGTM" or silence (`skip_comment`) is what this
repo's real history actually shows — don't manufacture verification detail beyond what a real PR
warrants.

## paulorsousa — approver on several 2026 PRs (#41, #43, #44, #45); bodies almost always empty

Four of paulorsousa's approvals in the sample carry no body at all. The one real exception is a short,
friendly **inline** question on PR#41 (adding a Docker Hub publish workflow), on `docker-build.sh`:

> "Just checking.. This script is not to be used on CI; it is more for 'local development,' right? 🙂"

This is the full extent of substantive paulorsousa review text found — a single clarifying question,
non-blocking (the PR was still approved), phrased gently with an emoji. Do not extrapolate a richer
paulorsousa "style" beyond "brief, friendly, asks one clarifying question when something's scope is
ambiguous" — that would go beyond what the record shows.

## ofekshenawa — one approval on record (PR#20), also the author of PR#19 above

Approved PR#20 with an empty body. No other review text from this account was found. Not enough
signal here for any voice beyond "approves."

## External, first-time contributors (fondoger #39, ruurdk #38, elicore #33, ikalchev-style
first-timers) — real pattern is long latency, not deep scrutiny

Real, dated examples: PR#38 (`ruurdk`, adding a Dockerfile) was opened 2024-10-03 and not merged until
2025-09-15 — eleven months later. PR#39 (`fondoger`, adding `--tls` flags) was opened 2025-04-09 and
merged 2025-09-15 — about five months later, and only after the author themselves pinged with
*"@filipecosta90 Please help review."* in an issue comment. Both eventually got a bare, empty-body
`APPROVED`. **The honest reading**: a long gap before merge in this repo's history reflects maintainer
bandwidth, not that the PR was under active scrutiny that whole time — don't imply a PR sitting open
for months was being carefully deliberated over unless there's an actual comment thread showing that.

## Issue triage is genuinely thin-to-absent

Real examples from the 7 issues surveyed: #37 (`ninuxer`, a real usability bug report about `-cmd` with
placeholders) has zero comments from anyone, maintainer or otherwise, and is still open. #13 (a TLS
connection-reset bug, two independent users corroborating with real repro commands and output) has no
maintainer reply in the thread. #16 and #23 are maintainer-filed issues with no further discussion.
#17 was closed by the reporter themselves with the comment *"no reply close"* — i.e., abandoned by the
reporter after getting no maintainer response, not resolved. **The honest baseline this establishes**:
do not have the triage bot imply that issues here typically get prompt maintainer engagement — they
often don't — and do not manufacture confidence that a "maintainer will follow up soon" beyond the
generic, honest framing every automated triage comment should carry regardless of project.

## Automated tooling: present, but no evidenced catch

`.github/workflows/codeql-analysis.yml` exists and presumably runs CodeQL on this repo, and
`build.yml` runs the Go test matrix (1.20.x/1.21.x) against a live Redis service container plus a
Codecov upload. However, unlike some other redis-performance repos, **no PR in the surveyed sample
shows a CodeQL alert, a Copilot review-bot comment, or an explicit Codecov-driven review discussion**.
Don't cite CodeQL or Copilot catches as real precedent for this repo the way you might for a project
with an evidenced history of them — there isn't one here yet.
