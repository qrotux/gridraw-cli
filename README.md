# gridraw

`gridraw` is a command-line client for a gridraw server. It
lists the grids a server publishes, prints a grid's descriptor, and queries
rows with a small filter language, writing the result in one of seven formats
that a shell pipeline can consume.

```console
$ gridraw from example where "rating >= 4" order "-rating" limit 3 -o csv
email,role,rating,price,isBanned,opensAt,createdAt,tags,locales,skills,interests
hank@example.com,admin,5,0.99,false,07:30:00,2026-08-20T16:11:20Z,"[""beta"",""early""]",...
ann@example.com,admin,4.8,19.90,false,09:00:00,2026-08-05T16:11:20Z,"[""vip"",""beta""]",...
dan@example.com,user,4.1,4.10,false,12:00:00,2026-08-30T16:11:20Z,"[""early"",""vip""]",...
```

Data goes to stdout, everything else — counts, progress, errors — goes to
stderr, so a redirect produces exactly the data you asked for.

## Contents

- [Install](#install)
- [Configuration](#configuration)
- [Listing grids: `list` and `grid`](#listing-grids-list-and-grid)
- [Querying rows: `from`](#querying-rows-from)
- [The `where` language](#the-where-language)
- [Output formats](#output-formats)
- [Paging and `--all`](#paging-and---all)
- [What goes to stderr](#what-goes-to-stderr)
- [Exit codes](#exit-codes)
- [The descriptor cache](#the-descriptor-cache)

## Install

Build from a checkout:

```console
$ go build -o bin/gridraw ./cmd/gridraw
$ ./bin/gridraw --help
```

Or install into `$(go env GOPATH)/bin`:

```console
$ go install github.com/qrotux/gridraw-cli/cmd/gridraw@latest
```

Three flags are accepted by every command:

| flag | meaning |
|---|---|
| `--config=NAME` | use this configuration profile instead of the current one |
| `--config-file=PATH` | read only this configuration file; discovery is cancelled |
| `-h`, `--help` | print the command's usage and exit |

`gridraw completion bash|zsh|fish|powershell` prints a shell completion script.

## Configuration

A configuration file holds named profiles — one per server you talk to — and
says which of them is current.

```yaml
current: default
configs:
  default:
    host: http://localhost:8080/api/grids
    header: "Bearer ${GRIDRAW_TOKEN}"
    defaultInfoOutput: yaml
    defaultDataOutput: csv
```

- `host` — the URL of the grids collection, without a trailing slash. Requests
  are built as `{host}/-/list`, `{host}/-/registry`, `{host}/{grid}` and
  `{host}/{grid}/rows`.
- `header` — the finished value of the `Authorization` header. Empty means the
  server is not guarded.
- `defaultInfoOutput` — `yaml` (the default) or `json`.
- `defaultDataOutput` — `csv` (the default), `tsv`, `json`, `jsona`, `jsonl`,
  `yaml` or `yamla`.

`${VAR}` in any string value is expanded from the environment when the file is
read, so a token need not be stored on disk. A variable that is not set is a
configuration error, not an empty string:

```console
$ gridraw list
Error: /home/me/.config/gridraw/config.yaml references unset environment variable(s): GRIDRAW_TOKEN
```

A `gridraw config` or `gridraw config use` that rewrites the file puts a
reference back as it was written, as long as the profile still holds the value
that reference resolved to. That covers every profile the command did not
touch, and the fields of an edited profile whose stored value you keep — an
empty answer to the token question stores `${GRIDRAW_TOKEN}` again, not the
token. Answer with a literal instead and the literal is what is written.

Files are written with mode `0600`.

### Which files are read

1. With `--config-file=PATH`, only that file is read and discovery is skipped.
   A missing file is an error.
2. Otherwise both of these are read when they exist:
   - the user file, `$XDG_CONFIG_HOME/gridraw/config.yaml`, or
     `~/.config/gridraw/config.yaml` when `XDG_CONFIG_HOME` is unset or empty.
     This path is the same on every platform, macOS included.
   - the local file, `./.gridraw.yaml` in the working directory.

A profile in the local file replaces the same-named profile from the user file
**whole** — fields are not merged one by one. `current` is taken from the local
file when it sets one, otherwise from the user file, otherwise it is `default`.
`--config=NAME` overrides `current` for one run. Asking for a profile that does
not exist is exit code 3.

### `gridraw config`

With no flags the command asks, in order: profile name, grids URL,
authorization (`bearer`, `basic` or `none`), default info output, default data
output, and which file to write. When you are editing an existing profile its
current values are offered as the defaults, and an empty answer keeps them.

Give the flags instead and it asks nothing. A `--host` together with
`--bearer-token` or `--basic-auth` is a complete profile, and every remaining
field takes its documented default:

```console
$ gridraw config --name=prod --host=https://grids.example.com/api/grids \
    --bearer-token="$TOKEN" --info-output=yaml --data-output=csv --global
Wrote profile "prod" to /home/me/.config/gridraw/config.yaml
```

| flag | meaning |
|---|---|
| `--name=NAME` | profile name; `default` when omitted |
| `--host=URL` | grids URL |
| `--bearer-token=TOKEN` | writes `header: Bearer TOKEN` |
| `--basic-auth=user:pass` | writes `header: Basic <base64(user:pass)>` |
| `--info-output=FORMAT` | `yaml` or `json` |
| `--data-output=FORMAT` | `csv`, `tsv`, `json`, `jsona`, `jsonl`, `yaml`, `yamla` |
| `--global` | write to the user file instead of `./.gridraw.yaml` |

`--bearer-token` and `--basic-auth` are mutually exclusive. Passing an empty
`--bearer-token=` writes no header, which is what an unguarded server wants.
A profile that already exists under the chosen name is overwritten.

Four subcommands read what is configured:

```console
$ gridraw config list
* example              http://localhost:8080/api/grids          /home/me/src/app/.gridraw.yaml

$ gridraw config use example
Current profile is "example" (/home/me/src/app/.gridraw.yaml)

$ gridraw config show          # or: config show NAME
name: example
host: http://localhost:8080/api/grids
header: Bearer ****cdef
defaultInfoOutput: yaml
defaultDataOutput: csv

$ gridraw config path
/home/me/src/app/.gridraw.yaml
```

`config list` marks the current profile with `*` and names the file each
profile came from. `config show` masks the credential — only the scheme and the
last four characters are printed. `config use` rewrites `current` in the
nearest file that was read, so a local file keeps its precedence.

## Listing grids: `list` and `grid`

| command | request |
|---|---|
| `gridraw list` | `GET {host}/-/list` |
| `gridraw grid` | `GET {host}/-/registry` |
| `gridraw grid NAME` | `GET {host}/{grid}` |

Both commands print in `defaultInfoOutput`, overridden by `-o/--output` with
`json` or `yaml`. A data format such as `csv` is rejected here.

`gridraw grid NAME` prints a **query view** of the descriptor: what someone —
or something — writing a query needs, and nothing that exists for a UI. The
operators are printed the way the `where` language accepts them, so they can be
copied straight into a filter; a test pins that every printed spelling parses
back to the operator it names.

The sample below adds titles and a description to show the shape; the demo
grid itself publishes none, and those keys are then absent.

```console
$ gridraw grid example
name: example
idColumn: id
pageSize: 10
defaultSort: createdAt desc
search: [Email, Tags]
columns:
    - key: email
      title: Email
      type: string
      description: Login email
      filters: [=, '!=', contains, not contains, starts with, ends with]
      sortable: true
      visible: true
    - key: role
      title: Role
      type: enum
      enum: {user: User, admin: Admin}
      filters: [in, not in]
      sortable: true
      visible: true
    - key: price
      title: Price
      type: decimal
      value: quoted, e.g. '19.99'
      filters: [=, '!=', '>', '>=', <, <=, between, not between]
      sortable: true
      visible: true
    - key: opensAt
      title: Opens at
      type: time
      value: HH:MM:SS, quoted
      step: 900
      filters: [=, '!=', '>', '>=', <, <=, between, not between]
      sortable: true
      visible: true
    - key: tags
      title: Tags
      type: string[]
      filters: [has any, has all, has only, not has any, is empty, is not empty]
      visible: true
    - key: prefs
      title: Preferences
      type: json
```

Per column: `title` and `description` as the server localises them, `type` (an
array column carries `[]`), `enum` as value → label, `value` when the literal
form is not obvious, `step` when a `time` or `datetime` column is coarser than
a second, `filters` in `where` spelling, `sortable` and `visible`. The grid's
own `description` sits next to its name. A key is omitted when it says nothing
— a column with no `filters` cannot be filtered at all, and `prefs` above is an
example — so a grid that publishes no descriptions shows none.
`skipTotal: true` appears only on a grid whose rows response carries no count.

`--raw` prints the server's descriptor instead, unchanged in `json` and
converted in `yaml`:

```console
$ gridraw grid example --raw -o json | head -c 120
{"name":"example","idColumn":"id","pageSize":10,"pageSizeOptions":[10,25,50,100],"defaultSort":{"column":"createdAt",...
```

`gridraw grid NAME` always goes to the server, past the cache, and stores what
it got — it reports the server's state and warms the cache for the next `from`.
`--no-cache` suppresses the write.

| flag | commands | meaning |
|---|---|---|
| `-o`, `--output=FORMAT` | `list`, `grid` | `yaml` or `json` |
| `--raw` | `grid NAME` | print the server's descriptor instead of the query view |
| `--no-cache` | `grid` | do not write the fetched descriptor into the cache |

## Querying rows: `from`

```
gridraw from GRID [columns "a,b"] [where "…"] [order "a,-b"] [search "…"] [limit N] [page M]
```

The keywords are positional, may come in any order, and each may appear at most
once. A repeated or unknown word is a usage error. Flags may sit before, after
or between the clauses, but never between a keyword and its value: a keyword
always takes the next word, so `order --quiet "-id"` would read `--quiet` as
the sort. A `--` ends the split: everything after it is read as tail words, so
`gridraw from -- users` names a grid that would otherwise look like a flag, and
`gridraw from users -- --quiet` is a usage error rather than a quiet run.

| keyword | effect |
|---|---|
| `columns "a,b"` | the columns to request and, for `csv`/`tsv`, their order |
| `where "…"` | the filter, see [below](#the-where-language) |
| `order "a,-b"` | the sort |
| `search "…"` | full-text search over the grid's searchable columns |
| `limit N` | page size, 1..100 |
| `page M` | which page to fetch, starting at 1 |

```console
$ gridraw from example columns "email,rating" where "rating >= 4" order "-rating" limit 2
Total: 3
email,rating
hank@example.com,5
ann@example.com,4.8
```

Without a `columns` clause the request asks for the descriptor's
`defaultVisible` columns in the order they are declared, which is also what the
text formats print. The id column is in the output only when the list names it,
even though the server returns it in every row.

`order` takes a comma-separated list. A leading `-` means descending, and the
long form works too, so `order "-rating"`, `order "rating desc"` and
`order "rating desc,email"` are all valid. At most 16 columns; a column the
descriptor does not mark sortable is a usage error. With no `order` clause the
server applies the grid's own `defaultSort`.

`search` requires a grid that declares searchable columns; otherwise it is a
usage error naming the grid.

### `from` flags

| flag | meaning |
|---|---|
| `-o`, `--output=FORMAT` | `json`, `jsona`, `jsonl`, `yaml`, `yamla`, `csv`, `tsv`; overrides `defaultDataOutput` |
| `--all` | page through the whole result |
| `--no-header` | omit the `csv`/`tsv` header row |
| `--null-val=STR` | text written for a null value in `csv`, `tsv`, `yaml`, `yamla` |
| `--quiet` | no `Total:` and no progress on stderr |
| `--progress` | print progress even when stderr is not a terminal |
| `--no-cache` | do not read or write the descriptor cache |
| `--refresh` | refetch the descriptor and update the cache |

## The `where` language

The same reference is in the binary: `gridraw help where`.

```
expr      := term ("or" term)*
term      := factor ("and" factor)*
factor    := "(" expr ")" | predicate
predicate := column op [value | "(" list ")" | value "and" value]
list      := value ("," value)*
```

`and` binds tighter than `or`, and parentheses work. Keywords and word
operators are case-insensitive; column names are spelled as in the descriptor.
There is no `not` in front of a group — use the negative operator instead, which
is what the error message suggests.

```console
$ gridraw from example where "rating >= 4 and isBanned = false and (role in ('admin') or email contains 'ann')" -o csv --quiet
email,role,rating,price,isBanned,opensAt,createdAt,tags,locales,skills,interests
hank@example.com,admin,5,0.99,false,07:30:00,2026-08-20T16:11:20Z,"[""beta"",""early""]",...
ann@example.com,admin,4.8,19.90,false,09:00:00,2026-08-05T16:11:20Z,"[""vip"",""beta""]",...
```

### Operators

| written | sent to the server |
|---|---|
| `=` | `eq` |
| `!=`, `<>` | `neq` |
| `>`, `>=`, `<`, `<=` | `gt`, `gte`, `lt`, `lte` |
| `~`, `contains` | `contains` |
| `!~`, `not contains` | `notContains` |
| `starts`, `starts with` | `starts` |
| `ends`, `ends with` | `ends` |
| `in (…)` | `in` |
| `not in (…)` | `notIn` |
| `between a and b` | `between` |
| `not between a and b` | `notBetween` |
| `is null` | `isNull` |
| `is not null` | `isNotNull` |
| `has any (…)` | `containsAny` |
| `has all (…)` | `containsAll` |
| `has only (…)` | `containsOnly` |
| `not has any (…)` | `notContainsAny` |
| `is empty` | `isEmpty` |
| `is not empty` | `isNotEmpty` |

The `and` inside `between a and b` belongs to the predicate and is consumed
before the boolean chain sees it, so `a between 1 and 2 and b = 3` means what it
looks like.

A column offers only the operators its descriptor publishes. Asking for another
one lists what it does offer:

```console
$ gridraw from example where "email > 'a'"
Error: column "email" of type string does not offer the operator gt at position 0: email > 'a'
^
it offers eq, neq, contains, notContains, starts, ends
```

### Literals

- a bare number — `4`, `-1.5` — is a JSON number;
- `true` and `false` are JSON booleans;
- `'…'`, `` `…` `` and `"…"` are strings; inside them `\'`, `` \` ``, `\"` and
  `\\` are the only escapes, and a backslash before anything else is a usage
  error — a Windows path is written `'C:\\new'`;
- a parenthesised list is for `in`, `not in` and the `has …` operators, and may
  not be empty.

A value is typed from how it is written, not from the column. The one silent
coercion is a JSON number written for a `decimal` column, which the server wants
as a string — this applies to list elements and to both bounds of a range too:

| written | column type | sent |
|---|---|---|
| `19.99` | `number` | `19.99` (number) |
| `19.99` | `decimal` | `"19.99"` (string — the one silent coercion) |
| `'19.99'` | `decimal` | `"19.99"` |
| `'19.99'` | `number` | usage error |
| `'true'` | `boolean` | usage error |

```console
$ gridraw from example where "rating = '4'"
Error: column "rating" is of type number, so "4" is not a valid value at position 0: rating = '4'
^
write a number, written without quotes
```

`uuid` values are lowercased. `enum` values are checked against the
descriptor's list, with the nearest name suggested on a typo. `date`, `time`
and `datetime` values are sent as written and validated by the server.

An unknown column is a usage error that suggests the closest name:

```console
$ gridraw from example where "nosuch = 1"
Error: grid "example" has no column "nosuch" at position 0: nosuch = 1
^
known columns: id, email, role, rating, price, isBanned, birthday, opensAt, createdAt, ...
```

### Limits

The filter is converted to disjunctive normal form by distributing `or` over
`and`, keeping the order in which the clauses were written. The server accepts
at most 10 groups of at most 20 clauses, and at most 16 sort columns. Exceeding
a limit is a usage error that reports the actual numbers and points out that
expanding parentheses multiplies the groups.

## Output formats

`-o/--output` overrides `defaultDataOutput`:

| format | what is printed |
|---|---|
| `json` | the response body as it came from the server |
| `jsona` | a JSON array of the rows |
| `jsonl` | one JSON object per line |
| `yaml` | `total:` and `rows:` as YAML; `hasPrev` and `hasNext` are dropped |
| `yamla` | a YAML array of the rows |
| `csv` | a table of the rows |
| `tsv` | the same, tab-separated |

```console
$ gridraw from example limit 1 -o jsonl --quiet
{"email":"eve@example.com","role":"admin","rating":2.7,"price":"9999.99","isBanned":false,...}

$ gridraw from example limit 1 -o yamla --quiet
- email: eve@example.com
  role: admin
  rating: 2.7
  price: "9999.99"
  isBanned: false
```

The columns of `csv` and `tsv`, and their order, are the `columns` clause when
there is one, otherwise the `defaultVisible` columns of the descriptor in
declaration order. An array or a `json` column is written as compact JSON in
its cell. `csv` quotes after RFC 4180: a field holding a comma, a quote or a
line break is wrapped in quotes and its inner quotes doubled. `tsv` never
quotes; it escapes instead, writing `\t`, `\n`, `\r` and `\\` for a tab, the two
line breaks and a backslash. Both end a record with CRLF. `--no-header` drops
the header row.

A `null` is an empty cell, or the text given by `--null-val`. That flag applies
to `csv`, `tsv`, `yaml` and `yamla`; in `json`, `jsona` and `jsonl` a null stays
a real `null`.

```console
$ gridraw from example limit 3 --null-val=NULL --quiet
email,role,rating,price,...
finn@example.com,user,NULL,12.50,...
```

## Paging and `--all`

Without `--all` exactly one request is made, for `page` (1 by default) with a
page size of `limit`, which defaults to the grid's own page size from the
descriptor.

With `--all` the CLI keeps fetching while the server reports another page.
`limit` then means the size of each page and defaults to 100; `page M` says
which page to start from, which is how an interrupted run is resumed.

```console
$ gridraw from example --all -o csv > rows.csv
Total: 8
```

Nothing is buffered. `csv` and `tsv` write the header once and then stream
rows; `jsonl` and `yamla` are streams by nature; `jsona` writes `[`, the rows
separated by commas, then `]`; and `json` writes `{"total": N, "rows": [`, the
rows, then `]}` — or the same envelope without `total` on a grid that skips it.
Note that with `--all` the `json` output is this streamed envelope rather than a
verbatim copy of one server response.

If a page request fails, the output is left unclosed on purpose — an interrupted
run must not look like a complete document — and the exit code is non-zero.
stderr then names the page and prints a command that continues from it:

```
Error: page 37: server returned 500: query failed
Resume with: gridraw from users where 'rating >= 4' -o csv --all page 37 --no-header
```

The hint carries no redirection — append it yourself with `>> rows.csv`.
Appending to the same file works cleanly for `csv` and `tsv` (with
`--no-header`), `jsonl` and `yamla`. For `json` and `jsona` a resumed run is a
separate valid document, and the hint says so instead.

**Rows added while you page can shift the result.** Paging is by page number,
not by cursor: a row inserted between two requests moves everything after it
down, so a row may be seen twice or missed. Sorting by a stable key and paging a
quiet table both reduce the risk, but nothing removes it.

## What goes to stderr

- `Total: {n}` on its own line after the first page. A grid that skips the
  total prints no such line. `--quiet` suppresses it.
- Progress, when stderr is a terminal: a redrawn line, `page 12/128 · 3000/31904
  rows`, or `page 12 · 3000 rows` when there is no total. `--progress` forces it
  on when stderr is not a terminal, printing one line per page instead of
  redrawing. `--quiet` suppresses both progress and `Total:`.
- Errors, prefixed with `Error: `. A `{"error": "…"}` body from the server is
  passed through unchanged.

```console
$ gridraw from example --all --progress -o csv limit 3 > rows.csv
Total: 8
page 1/3 · 3/8 rows
page 2/3 · 6/8 rows
page 3/3 · 8/8 rows
```

## Exit codes

| code | meaning |
|---|---|
| 0 | success |
| 1 | a general or network failure |
| 2 | a usage error: an unparseable `where`, an unknown column or operator, bad arguments |
| 3 | the configuration is missing or invalid |
| 4 | the server answered 4xx |
| 5 | the server answered 5xx |

```console
$ gridraw from example where "nosuch = 1"; echo "exit=$?"
Error: grid "example" has no column "nosuch" ...
exit=2

$ gridraw list --config=absent; echo "exit=$?"
Error: no configuration named "absent"; have example
exit=3

$ gridraw grid nosuchgrid; echo "exit=$?"
Error: server returned 404: unknown grid
exit=4
```

## The descriptor cache

`from` needs a grid's descriptor to validate the columns, operators and values
before it sends anything, so descriptors are cached for **5 minutes** under the
user cache directory:

```
{cache}/gridraw/{profile}/{grid}.json
```

`{cache}` is `$XDG_CACHE_HOME` when set, `~/.cache` otherwise, on every
platform. The profile name is part of the path, so a `prod`
and a `dev` grid of the same name never share an entry.

| flag | effect |
|---|---|
| `--refresh` | refetch the descriptor and overwrite the cached copy |
| `--no-cache` | neither read nor write the cache |

A cache file that is corrupt or unreadable is not an error: it is refetched. The
cache is filled by both commands that need a descriptor — `from`, on a miss or
with `--refresh`, and `gridraw grid NAME`, which always fetches.

```console
$ gridraw from example --refresh limit 1 -o csv --quiet
email,role,rating,price,...
```

## License

See [LICENSE](LICENSE).
