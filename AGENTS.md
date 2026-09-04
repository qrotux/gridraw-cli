# Using the gridraw CLI as an agent

Instructions for an LLM agent that queries gridraw servers through the `gridraw`
binary. This describes how to *operate the tool*; for working on the tool's own
source, read [CLAUDE.md](CLAUDE.md).

## The order of operations

1. `gridraw list` — the grids this server publishes. Names come from here; do
   not guess one.
2. `gridraw grid NAME -o json` — the query view of that grid. Read it before
   writing any filter. It carries every column's key, type, allowed operators
   and enum values.
3. `gridraw from NAME … -o jsonl` — the query.

Never write a `where` clause from memory of a schema. The descriptor is the
authority, it is cached for five minutes, and reading it costs one request.

## Reading the query view

```yaml
columns:
    - key: role            # what a query spells
      title: Role          # a display label; never use it in a query
      type: enum
      enum: {user: User, admin: Admin}   # value: label — filter on the LEFT side
      filters: [in, not in]              # the only operators this column accepts
      sortable: true                     # may appear in `order`
      visible: true                      # in the default column set
```

Rules that follow from it:

- **`filters` is exhaustive.** An operator not listed is refused locally. The
  spellings are exactly what the filter language accepts — copy them verbatim,
  including the multi-word ones (`not in`, `is not null`, `has any`).
- **A column with no `filters` cannot be filtered at all**, whatever its type.
- **A missing `sortable` or `visible` means false.** Absent keys are false or
  empty, never unknown.
- **`type` ending in `[]` is an array column**: it takes only `has any`,
  `has all`, `has only`, `not has any`, `is empty`, `is not empty`.
- **`value:` is the literal form** for the types whose spelling is not obvious.
- **`step:` is seconds** on a `time` or `datetime` column. A value must be
  aligned to it — with `step: 900`, `'09:15:00'` is accepted and `'09:07:00'`
  is refused *by the server*, which the CLI cannot catch for you.
- **`search:` names the columns quick search covers.** `search "text"` takes
  free text; those names are not usable as filter columns.
- **`skipTotal: true`** means the rows response carries no count: page with
  `--all`, and expect no `Total:` line.

## Writing a filter

```
gridraw from users where "rating >= 4 and (role in ('admin') or email contains 'ann')"
```

`and` binds tighter than `or`; parentheses work; there is no `not` before a
group — use the negative operator. Full grammar and the operator table:
`gridraw help where`.

Literals are typed by how they are written, not by the column:

| you write | it is |
|---|---|
| `4`, `-1.5` | a number |
| `true`, `false` | a boolean |
| `'x'`, `` `x` ``, `"x"` | a string |
| `(a, b)` | a list, for `in`, `not in` and the `has …` operators |

Consequences worth memorising:

- A `string`, `date`, `time`, `datetime`, `uuid` or `enum` value **must be
  quoted**. `role in (admin)` is an error; `role in ('admin')` is right.
- A `number` column needs a bare number: `rating = '4'` is an error.
- A `boolean` column needs bare `true`/`false`: `isBanned = 'true'` is an error.
- A `decimal` column wants a string, and a bare number is converted for you, so
  `price >= 19.99` and `price >= '19.99'` are both correct.
- Inside a string only `\'`, `` \` ``, `\"` and `\\` are escapes; any other
  backslash is an error, so a Windows path is `'C:\\new'`.

Limits, enforced locally: at most 10 OR groups of at most 20 conditions —
parentheses multiply the groups — at most 16 sort columns, and `limit` between
1 and 100.

## Getting output you can parse

- **`-o jsonl`** for row-by-row work: one compact JSON object per line, streamed.
- **`-o jsona`** when you want one array.
- **`-o json`** returns the server's response, `{"rows": …, "total": …,
  "hasPrev": …, "hasNext": …}`; under `--all` it is an envelope of the same
  shape.
- **`--all`** pages through everything; `limit` sets the page size.
- **`--quiet`** silences `Total:` and progress so stderr stays empty.

stdout carries data only, on every path — a redirect never picks up a
diagnostic. Read the payload from stdout and treat stderr as human text.

## Reacting to failure

Branch on the exit code, not on the message text:

| code | meaning | what to do |
|---|---|---|
| 0 | success | — |
| 1 | network or other failure | retry once, then report |
| 2 | usage: bad `where`, unknown column or operator, bad arguments | re-read the descriptor and rewrite the query; do not retry unchanged |
| 3 | configuration missing or invalid | stop and ask a human |
| 4 | the server answered 4xx | the request was refused; fix it, do not retry unchanged |
| 5 | the server answered 5xx | retry with backoff |

A usage error names the offending fragment, its position, and often the fix
("did you mean …", "it offers …"). Use it to rewrite the query rather than
guessing again.

When `--all` fails midway, stdout holds a partial, deliberately unclosed
document, and stderr carries the exact command that resumes the run from the
failed page. Take that command as given; do not reconstruct it.

## Things that will waste your turns

- Filtering on a column's `title` instead of its `key`.
- Using `=` on an `enum` column — it takes `in ('value')`.
- Omitting quotes around a date, time or uuid.
- Sending `-o csv` to `list` or `grid`; those two print `yaml` or `json` only.
- Assuming a column is sortable, filterable or nullable without checking
  `filters` and `sortable`.
- Parsing the human-readable error text instead of the exit code.
