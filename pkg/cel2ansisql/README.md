# CEL2ANSISQL

This package builds a mapper between the [Common Expression
Language](https://cel.dev/) and [Structured Query
Language](https://en.wikipedia.org/wiki/SQL) standards. Specifically, this
package attempts to provide a mapping for ANSI SQL, as a way to ensure the highest re-usability of the SQL generated.


## Mapping Table

### Operators

| CEL  | SQL |
| ---- | --- |
| ==   | =   |
| &&   | AND |
| \|\| | OR  |

### Functions

| CEL              | SQL                     |
| ---------------- | ----------------------- |
| `startsWith($1)` | `LIKE CONCAT($1, '%%')` |