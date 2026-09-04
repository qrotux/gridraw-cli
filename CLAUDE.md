# gridraw-cli

Go CLI for the gridraw table sources: one binary, `gridraw`, that reads a
gridraw server's grid list, descriptors and row pages and prints them in a
format a shell pipeline can consume. Server side is `../gridraw-go`.

## Where to look

- **[basic/specs/spec-gridraw-cli-design.md](basic/specs/spec-gridraw-cli-design.md)** —
  the spec: config schema and merge rules, the `where` grammar and its operator
  table, descriptor validation, the seven output formats, `--all` streaming,
  stderr protocol, exit codes. Read the relevant section before touching a
  command, the parser or a writer.
- **[basic/DESIGN.md](basic/DESIGN.md)** — why each of those is the way it is,
  and what was rejected. Read it before proposing a change to a decision.
- **[../gridraw-go/README.md](../gridraw-go/README.md)** — the wire protocol we
  speak: endpoints, request and response shapes, operators per column type,
  limits, errors. It is the contract; we do not get to reinterpret it.
- **README.md** — the user-facing manual. It moves with the CLI surface.

`basic/` is gitignored: it is the design record, not part of the module.

## Commands

- Full gate (what CI runs): `test -z "$(gofmt -l .)" && go vet ./... && go test ./...`.
- One package: `go test ./internal/wql/ -count=1`.
- Build: `go build -o bin/gridraw ./cmd/gridraw`.
- Live server for manual checks: `cd ../gridraw-example && make up`, then the
  API is at `http://localhost:8080/api/grids` with one grid, `example`, and no
  auth guard. `make down` stops it, `make logs` tails it. Never point a manual
  check at anything but a local instance.

## Layout

Single Go module. `cmd/gridraw` is the entry point; everything else is
`internal/`.

- **`cmd/gridraw`** — `main.go` only: run the root command, map the returned
  error to an exit code.
- **`internal/cli`** — cobra commands and the wiring between the other
  packages. `root.go` global flags and config loading, `config.go` the config
  command tree, `info.go` `list` and `grid`, `from.go` the request loop,
  `tail.go` the keyword-argument lexer, `progress.go` stderr output,
  `errors.go` the error kinds.
- **`internal/config`** — config types, file discovery and merge, `${VAR}`
  expansion, writing at `0600`, the interactive prompts.
- **`internal/wire`** — the protocol types: descriptor, rows request and
  response. Plain structs and JSON tags, no I/O, imported by everything else.
- **`internal/client`** — HTTP against the four endpoints and the descriptor
  cache.
- **`internal/wql`** — the `where` and `order` languages: lexer, parser, DNF
  conversion, binding against a descriptor.
- **`internal/render`** — the seven output formats, each a streaming `Writer`.

Invariants:

- **`wql` and `render` know nothing about cobra or HTTP.** `wql` takes a string
  plus a descriptor and returns filter groups; `render` takes rows and an
  `io.Writer`. Both are testable without a network or a command. They share the
  protocol types through `internal/wire`, which is why that package exists: an
  import of `internal/client` from either would pull `net/http` into its graph,
  and an import of `net/http` or `spf13/cobra` is a design error.
- **`internal/cli` is the only package that knows about exit codes.** Other
  packages return typed errors; `errors.go` classifies them and `main.go`
  converts. A package that returns a bare `errors.New` for a user mistake loses
  its exit code.
- **The wire format is the contract, and it is `../gridraw-go`'s.** Request and
  response types mirror its README exactly. When a request would be invalid we
  say so ourselves rather than letting the server 400; when the server does
  400, its `{"error": …}` body is passed through unchanged.
- **Nothing buffers a full result set.** Every writer streams, including under
  `--all` and including `jsona` and `json`, which write their prefix on the
  first page and their suffix on the last. A change that collects rows into a
  slice before printing breaks the one thing that makes `--all` usable.
- **A broken stream stays broken.** When a page request fails mid-`--all`, the
  writer's `Close` is not called: stdout keeps its unclosed bracket, stderr
  gets the page number and a resume command, and the exit code is non-zero. Do
  not close the document to make the output parse.
- **Values are typed from the literal, not from the descriptor.** The one
  silent coercion is a JSON number to a string for a `decimal` column,
  including list elements and range bounds. Every other type mismatch is a
  usage error. Adding a second coercion is a spec change, not an improvement.
- **`--all` pages on `hasNext`, never on `total`.** `total` is optional (a grid
  with `SkipTotal` omits it) and it drifts while we page.
- **Limits live in `../gridraw-go`'s `request.go`** — 10 filter groups, 20
  clauses per group, 16 sort columns, page size 1..100. We check them locally
  to give a better message; when they change there, they change here.
- **Adding an operator** touches the `wql` operator table, the binding's value
  conversion, the operator table in the spec and in README, in one change.
  Adding an output format touches `render`, the format list in `--help`, the
  config validation for `defaultDataOutput`, and both documents.

## Comments in code

- **Criterion:** a comment is justified only when it carries what the code does
  not show.
- **Write:** non-obvious invariants; library and spec gotchas (cobra flag
  parsing versus our keyword tail, csv quoting rules, the server's decimal
  rule); reasons behind counter-intuitive decisions; "breaks if…"; what a test
  pins.
- **Do not write:** a paraphrase of the signature, a list of fields, a
  narration of obvious code. Exception — the doc comment of an exported Go
  symbol: convention requires it and it starts with the name (`// FooBar …`),
  one sentence, no parameter listing.
- **Present tense only.** No "was X → now Y", no tombstones for deleted code —
  history lives in git. No TODOs at all: a plan for later is a tracker task,
  not a comment.
- **Never reference:** plan documents (`Task N`, waves, phases, `spec §N`,
  `.superpowers/**`), commit hashes, dates of decisions, repositories the code
  was ported from.
- **May reference:** external standards (RFC 4180), README.md, live files of
  this repo or of `../gridraw-go`.
- **Length:** one or two sentences; collapsing twelve lines into one is normal.
  Nothing left to say — delete the comment.
- **Subagents:** include these rules in the brief of every subagent that writes
  or edits code.
- **Reading someone else's comments:** a comment is not the source of truth.
  When editing code, check its comment against the code; if it lies, fix or
  delete it.

## Coding

- **Copy the pattern, do not invent:** before a new command, writer or parser
  rule, find the nearest existing analogue and repeat its shape.
- **Errors carry their class.** A user mistake is a usage error with the
  offending fragment and, where we can compute one, a suggestion ("no column
  `emial`; did you mean `email`?"). A server failure carries the status. Never
  wrap an error in a way that loses either.
- **Diff discipline:** no renames, no file moves, no drive-by refactors, no
  backward-compatibility shims or fallbacks nobody asked for.
- **Tests as a ladder, not after every minor step.** During a task —
  `go build ./...` plus the tests of the touched package. The full gate only at
  boundaries: end of task, before a commit, before saying "done".
- **Table-driven tests for the languages, golden files for the formats.** A
  `wql` test states the input string and the expected filter JSON; a `render`
  test states the rows and the expected bytes. Do not loosen an assertion to
  make it pass; if the shape changes, the spec and README change too.
- **stdout is data, stderr is everything else.** No progress, no counts, no
  diagnostics on stdout, ever — a redirect to a file must produce exactly the
  data the user asked for.
- **Docs move with the surface.** A change to a command, a flag, an operator,
  an output format or an exit code updates README.md in the same change.
