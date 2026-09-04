# gridraw — recipes

Task-oriented notes for people who already have the binary. The reference —
every flag, the full operator table, the exact semantics — is in
[README.md](README.md), and the filter language is also in the binary:
`gridraw help where`.

## Set up a server

```sh
gridraw config --name=prod --host=https://example.com/api/grids --bearer-token="$TOKEN"
```

Without flags the command asks instead, and offers the current values when the
profile already exists. It writes `./.gridraw.yaml` by default, or the user file
with `--global`. Keep the token out of the file by writing a reference:

```yaml
configs:
  prod:
    host: https://example.com/api/grids
    header: "Bearer ${GRIDRAW_TOKEN}"
```

`${VAR}` is expanded when the file is read and written back as the reference, so
editing a profile never bakes the secret onto disk. An unset variable is an
error, not an empty header.

Several servers at once: one profile each, `--config=prod` per command or
`gridraw config use prod` to switch. `gridraw config list` shows which is
current and which file each came from.

## Find out what you can query

```sh
gridraw list              # the grids this server publishes
gridraw grid              # the same, with each grid's columns
gridraw grid users        # one grid: types, operators, enum values
```

`gridraw grid NAME` prints only what a query needs. Each column's `filters`
lists its operators **in the spelling the filter language accepts**, so they can
be copied straight into a `where` clause. `--raw` prints the server's own
descriptor instead.

## Query rows

```sh
gridraw from users limit 10
gridraw from users columns "email,role" where "rating >= 4" order "-rating" limit 20
gridraw from users where "role in ('admin','mod') and lastSeenAt is not null"
gridraw from users search "ivan" limit 5
```

The clauses are keywords, not flags, and may come in any order: `columns`,
`where`, `order`, `search`, `limit`, `page`. A keyword always takes the next
word, so never put a flag between a keyword and its value.

`order "-rating"` sorts descending; `order "rating desc"` is the same thing.

## Export the whole table

```sh
gridraw from users -o csv --all > users.csv
```

`Total:` and a progress line go to stderr, so the file holds only data. If a
page fails, the run stops and prints the command that continues it:

```
Error: page 37: server returned 500: query failed
Resume with: gridraw from users -o csv --all page 37 --no-header
```

Append that to the same file with `>> users.csv`. Appending works for `csv`,
`tsv`, `jsonl` and `yamla`; `json` and `jsona` produce one closed document, so a
continuation is a separate file.

## Feed another program

```sh
gridraw from users -o jsonl --all --quiet | while read -r row; do …; done
gridraw from users -o json limit 1 | jq '.rows[0].email'
gridraw from users -o jsona --all | jq 'length'
gridraw grid users -o json | jq '.columns[] | select(.type=="enum") | .key'
```

`jsonl` is the one to parse row by row — it streams, one object per line, and
two runs concatenate. `--quiet` silences `Total:` and the progress line when
even stderr should stay clean.

## Formats at a glance

| format | what you get |
|---|---|
| `csv`, `tsv` | the rows as a table; `--no-header` omits the header |
| `jsonl` | one compact object per line |
| `jsona` | a JSON array of the rows |
| `json` | the server's response as it came (an envelope with the total under `--all`) |
| `yaml` | the total and the rows as YAML |
| `yamla` | a YAML sequence of the rows |

`--null-val=NULL` writes something visible for a null cell in `csv`, `tsv`,
`yaml` and `yamla`; by default the cell is empty.

## In a script

```sh
if ! rows=$(gridraw from users where "role in ('admin')" -o jsonl --quiet); then
  case $? in
    2) echo "the query is wrong" >&2 ;;
    3) echo "no configuration" >&2 ;;
    4) echo "the server refused it" >&2 ;;
    5) echo "the server failed; retry" >&2 ;;
    *) echo "network or something else" >&2 ;;
  esac
  exit 1
fi
```

stdout carries data and nothing else on every path, so redirecting it is always
safe.

## When something is refused

Most mistakes are caught before a request goes out, with the position and a
suggestion:

```console
$ gridraw from users where "emial = 'a'"
Error: grid "users" has no column "emial" at position 0: emial = 'a'
^
did you mean "email"?

$ gridraw from users where "role = 'admin'"
Error: column "role" of type enum does not offer the operator eq at position 0: role = 'admin'
^
it offers in, not in
```

An enum takes `in ('admin')`, not `=`. Money is a `decimal`: `price >= 19.99`
works because the number is quoted for you, and `price >= '19.99'` is the
explicit form. A `time` or `datetime` column may have a `step` — with `step:
900` a value must land on a quarter hour, and only the server can refuse it.

Descriptors are cached for five minutes; `--refresh` refetches, `--no-cache`
ignores the cache entirely.
