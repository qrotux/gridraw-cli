package cli

import "github.com/spf13/cobra"

// whereHelp is the reference for the filter language, printed by
// `gridraw help where`. It repeats README's "The where language" section; the
// two move together.
const whereHelp = `The where clause of ` + "`gridraw from`" + ` is an SQL-like expression that the CLI
parses locally, validates against the grid descriptor and sends as the server's
disjunctive normal form.

  expr      := term ("or" term)*
  term      := factor ("and" factor)*
  factor    := "(" expr ")" | predicate
  predicate := column op [value | "(" list ")" | value "and" value]
  list      := value ("," value)*

"and" binds tighter than "or", and parentheses work. Keywords and word
operators are case-insensitive; column names are spelled as in the descriptor.
There is no "not" in front of a group — use a negative operator instead.

OPERATORS

  written                  sent to the server
  =                        eq
  !=  <>                   neq
  >  >=  <  <=             gt  gte  lt  lte
  ~   contains             contains
  !~  not contains         notContains
  starts, starts with      starts
  ends, ends with          ends
  in (…)                   in
  not in (…)               notIn
  between a and b          between
  not between a and b      notBetween
  is null                  isNull
  is not null              isNotNull
  has any (…)              containsAny
  has all (…)              containsAll
  has only (…)             containsOnly
  not has any (…)          notContainsAny
  is empty                 isEmpty
  is not empty             isNotEmpty

The "and" inside "between a and b" belongs to the predicate, so
"a between 1 and 2 and b = 3" means what it looks like.

A column offers only the operators its descriptor publishes; asking for another
one lists what it does offer. Run "gridraw grid NAME" to see them.

LITERALS

  4, -1.5              a JSON number
  true, false          a JSON boolean
  'x'  ` + "`x`" + `  "x"        a string
  (a, b, c)            a list, for in / not in / has …; may not be empty

Inside a string, \' \` + "`" + ` \" and \\ are the only escapes; a backslash before
anything else is a usage error, so a Windows path is written 'C:\\new'.

A value is typed from how it is written, not from the column. The one silent
coercion is a JSON number written for a decimal column, which the server wants
as a string — this applies to list elements and to both bounds of a range:

  19.99     for a number column    → 19.99 (number)
  19.99     for a decimal column   → "19.99" (string, the one coercion)
  '19.99'   for a decimal column   → "19.99"
  '19.99'   for a number column    → usage error
  'true'    for a boolean column   → usage error

uuid values are lowercased. enum values are checked against the descriptor's
list, with the nearest name suggested on a typo. date, time and datetime values
are sent as written and validated by the server.

LIMITS

The filter is converted to disjunctive normal form by distributing "or" over
"and". The server accepts at most 10 groups of at most 20 clauses. Exceeding a
limit is a usage error reporting the actual numbers, since expanding
parentheses multiplies the groups.

EXAMPLES

  gridraw from users where "rating >= 4"
  gridraw from users where "email contains 'ann' and isBanned = false"
  gridraw from users where "role in ('admin','mod') or rating between 4 and 5"
  gridraw from users where "lastSeenAt is null"
  gridraw from users where "tags has any ('go','cli')"
  gridraw from users where "balance >= 19.99 and (plan = 'pro' or plan = 'team')"
`

// newWhereTopic returns the `where` help topic: a command with no Run, which
// cobra lists under "Additional help topics".
func newWhereTopic() *cobra.Command {
	return &cobra.Command{
		Use:   "where",
		Short: "The filter language used by `gridraw from ... where \"...\"`",
		Long:  whereHelp,
	}
}
