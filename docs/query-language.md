# Query Language (TSL)

kubectl-debug-queries supports an optional `--query` / `-q` flag on all commands that output structured data (`get`, `list`, `events`, `logs`). The query uses **TSL (Tree Search Language)**, an open-source, SQL-like query language for filtering structured data.

## Query Structure

```
[SELECT fields] [WHERE condition] [ORDER BY field [ASC|DESC]] [LIMIT n]
```

All clauses are optional. The most common form is a bare `WHERE` filter:

```
where <condition> [order by <field> [asc|desc]] [limit <n>]
```

If the query does not start with a recognized keyword (`select`, `where`, `order by`, `sort by`, `limit`), `where` is automatically prepended. This means you can write short-form queries:

```bash
# These are equivalent:
--query "Status = 'Running'"
--query "where Status = 'Running'"

# Short-form works with any expression:
--query "Name ~= 'nginx-.*'"
--query "level = 'ERROR' or level = 'WARN'"
--query "fields.map is not null"
```

### Clause Reference

| Clause | Required | Description |
|--------|----------|-------------|
| `SELECT` | No | Choose specific fields (affects JSON/YAML output only) |
| `WHERE` | No | Filter rows by condition |
| `ORDER BY` / `SORT BY` | No | Sort by field, optionally `ASC` (default) or `DESC` |
| `LIMIT` | No | Maximum number of results to return |

### Examples

```sql
-- Filter only
where Status = 'Running'

-- Filter with sorting
where Status = 'Running' order by Name

-- Filter with sorting and limit
where Restarts > 0 order by Restarts desc limit 10

-- Select specific fields (JSON/YAML output)
select Name, Status where Restarts > 0

-- Full query
select Name, Status, Restarts where Restarts > 0 order by Restarts desc limit 5
```

## Output Format Behavior

The `SELECT` clause interacts differently with each output format:

| Format | SELECT behavior | Columns / fields shown |
|--------|----------------|------------------------|
| `table` | Ignored | Original server-side columns (always) |
| `markdown` | Ignored | Original server-side columns (always) |
| `json` | Applied | Only selected fields |
| `yaml` | Applied | Only selected fields |

`WHERE`, `ORDER BY`, and `LIMIT` apply to **all** output formats.

## Data Types and Literals

### Strings

Strings are enclosed in single quotes:

```sql
Name = 'my-pod'
Status = 'Running'
```

### Numbers

Integer and decimal literals:

```sql
Restarts = 0
Restarts > 5
```

### SI Unit Suffixes

SI suffixes express byte quantities concisely (expanded to plain numbers at evaluation time):

| Suffix | Multiplier | Example | Expanded |
|--------|-----------|---------|----------|
| `Ki` | 1,024 | `4Ki` | 4096 |
| `Mi` | 1,048,576 | `512Mi` | 536870912 |
| `Gi` | 1,073,741,824 | `4Gi` | 4294967296 |
| `Ti` | 1,099,511,627,776 | `1Ti` | 1099511627776 |

### Booleans

```sql
parsed = true
previous = false
```

### Arrays (for `IN` operator)

Arrays use square brackets with single-quoted elements:

```sql
Status in ['Running', 'Pending']
Type in ['Warning', 'Normal']
```

### Null

```sql
logger is null
source is not null
```

## Operators

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equal to | `Status = 'Running'` |
| `!=`, `<>` | Not equal to | `Status != 'Failed'` |
| `<` | Less than | `Restarts < 3` |
| `<=` | Less than or equal | `Restarts <= 1` |
| `>` | Greater than | `Restarts > 5` |
| `>=` | Greater than or equal | `Restarts >= 10` |

### String Matching Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `like` | Case-sensitive pattern (`%` = any, `_` = one char) | `Name like 'nginx-%'` |
| `ilike` | Case-insensitive LIKE | `Name ilike '%WEB%'` |
| `~=` | Regular expression match | `Name ~= 'nginx-[a-z]+'` |
| `~!` | Regular expression does not match | `Name ~! 'test-.*'` |

#### Regular Expression Notes

The `~=` and `~!` operators accept full regular expressions:

```sql
-- Anchored patterns
Name ~= '^web-[0-9]+$'

-- Alternation
Name ~= '^(web|app|db)-.*'

-- Case-insensitive regex
Message ~= '(?i)error.*timeout'
```

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `Status = 'Running' and Restarts > 0` |
| `or` | Logical OR | `Type = 'Warning' or Type = 'Error'` |
| `not` | Logical NOT | `not (Status = 'Succeeded')` |

Parentheses control precedence:

```sql
(Status = 'Running' or Status = 'Pending') and Restarts > 0
not (Type = 'Normal')
```

### Set and Range Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `in` | Value in array | `Status in ['Running', 'Pending']` |
| `not in` | Value not in array | `Reason not in ['Pulled', 'Started']` |
| `between ... and` | Inclusive range | `Restarts between 1 and 10` |

**Important**: The `in` and `not in` operators require **square brackets** `[...]`, not parentheses.

### Null Checking Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `is null` | Field is null | `logger is null` |
| `is not null` | Field is not null | `source is not null` |

### Arithmetic Operators

| Operator | Description |
|----------|-------------|
| `+` | Addition |
| `-` | Subtraction |
| `*` | Multiplication |
| `/` | Division |
| `%` | Modulo |

## SELECT, ORDER BY, and LIMIT

### SELECT Clause

The optional `SELECT` clause limits which fields appear in JSON and YAML output:

```sql
-- Select specific fields
select Name, Status where Restarts > 0

-- Field with alias
select Name, Restarts as restart_count where Restarts > 0

-- Reducers: sum, len, any, all (for nested array data)
select Name, len(fields) as field_count where parsed = true
```

### ORDER BY Clause

Sort results by field in ascending (default) or descending order:

```sql
-- Ascending (default)
where Status = 'Running' order by Name

-- Descending
where Restarts > 0 order by Restarts desc

-- Multiple sort keys
where Status = 'Running' order by Status, Name asc
```

### LIMIT Clause

Restrict the number of results:

```sql
-- Top 10
where Status = 'Running' order by Name limit 10

-- First 5 matching
where Restarts > 0 limit 5
```

## Field Access

### Simple Fields

For resources (`get`, `list`, `events`), field names correspond to the server-side table columns you see in table output (e.g. `Name`, `Status`, `Ready`, `Age`, `Restarts`, `Type`, `Reason`, `Message`).

For logs (`--output json` or `--output smart`), the queryable fields are: `timestamp`, `level`, `message`, `source`, `logger`, `raw_line`, `format`, `parsed`.

### Column Names with Spaces

Some Kubernetes columns contain spaces (e.g. "Last Seen", "Nominated Node", "Readiness Gates"). In queries, replace spaces with **underscores**. The matching is case-insensitive:

```sql
-- "Last Seen" column
where Last_Seen = '<unknown>'

-- "Nominated Node" column
where Nominated_Node is not null
```

### Dot Notation (Nested Fields)

Access nested fields with dots (useful for log entry `fields`):

```sql
-- Log entry extra fields
where fields.request_id = 'abc-123'
```

### Array Index and Wildcard Access

```sql
-- Specific index
fields[0].key = 'value'

-- Wildcard (all elements)
any(fields[*].key = 'value')
```

## Functions

### `len(field)`

Returns the length of an array or string:

```sql
len(fields) > 2
len(Name) > 10
```

### `sum(field)`

Sums numeric values across array elements:

```sql
sum(fields[*].count) > 100
```

### `any(condition)` / `all(condition)`

Test array elements against a condition:

```sql
any(fields[*].key = 'error')
all(fields[*].status = 'ok')
```

## kubectl-debug-queries Examples

### Resources (list, get)

The `where` keyword is optional -- bare expressions work as shorthand.

```bash
# List only Running pods (bare expression, "where" auto-prepended)
kubectl debug-queries list --resource pods --namespace default --query "Status = 'Running'"

# Same thing with explicit "where"
kubectl debug-queries list --resource pods --namespace default --query "where Status = 'Running'"

# Regex match on pod names
kubectl debug-queries list --resource pods --namespace default --query "Name ~= 'nginx-.*'"

# Pods with restarts, sorted descending
kubectl debug-queries list --resource pods --namespace default \
  --query "where Restarts > 0 order by Restarts desc"

# Top 5 pods by restart count (JSON with field selection)
kubectl debug-queries list --resource pods --namespace default --output json \
  --query "select Name, Restarts where Restarts > 0 order by Restarts desc limit 5"

# Deployments not available
kubectl debug-queries list --resource deployments --all-namespaces --query "Ready ~= '0/.*'"

# Nodes that are not Ready
kubectl debug-queries list --resource nodes --namespace default --query "Status != 'Ready'"

# Get a single resource with field selection
kubectl debug-queries get --resource pod --name my-pod --namespace default --output json \
  --query "select Name, Status"
```

### Events

```bash
# Warning events only (bare expression)
kubectl debug-queries events --namespace default --query "Type = 'Warning'"

# BackOff events sorted by last seen
kubectl debug-queries events --namespace default \
  --query "Reason = 'BackOff' order by Last_Seen desc"

# Warning events with field selection
kubectl debug-queries events --namespace default --output json \
  --query "select Reason, Message where Type = 'Warning'"

# Events matching a pattern in the message
kubectl debug-queries events --all-namespaces --query "Message ~= '(?i)timeout'"

# Events with a specific reason, limited
kubectl debug-queries events --namespace default \
  --query "Reason in ['Failed', 'BackOff', 'Unhealthy'] limit 20"
```

### Logs

The `--query` flag works on parsed log fields: `timestamp`, `level`, `message`, `source`, `logger`, `raw_line`, `format`, `parsed`, and any nested key under `fields` (the extra key=value pairs extracted from structured logs).

The `where` keyword is optional -- bare expressions are automatically treated as filters.

```bash
# Only ERROR log entries (smart format)
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 200 \
  --query "where level = 'ERROR'"

# Short-form (bare expression, "where" auto-prepended)
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 200 \
  --query "level = 'ERROR'"

# Errors and warnings
kubectl debug-queries logs --name my-pod --namespace default --tail 500 \
  --query "level = 'ERROR' or level = 'WARN'"

# Regex match on message
kubectl debug-queries logs --name my-pod --namespace default --tail 200 \
  --query "message ~= '(?i)timeout|connection refused'"

# Filter by nested field (extra key=value pairs from structured logs)
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 100 \
  --query "fields.map is not null"

# Logs that mention a specific plan
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 200 \
  --query "fields.plan = 'migrate-rhel8'"

# Combine nested field check with level
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 500 \
  --query "level = 'ERROR' and fields.plan is not null"

# Logs from a specific source file
kubectl debug-queries logs --name my-pod --namespace default --tail 200 \
  --query "source ~= 'reconcile.*'"

# JSON output with field selection
kubectl debug-queries logs --name my-pod --namespace default --tail 200 --output json \
  --query "select timestamp, level, message where level = 'ERROR'"

# JSON output: only entries that have extra fields
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 100 --output json \
  --query "select timestamp, level, message, fields where fields is not null"
```

### MCP Server (debug_read)

```json
{"command": "list", "flags": {"resource": "pods", "namespace": "default", "query": "where Status = 'Running'"}}
{"command": "list", "flags": {"resource": "pods", "namespace": "default", "output": "json", "query": "select Name, Status where Restarts > 0"}}
{"command": "events", "flags": {"namespace": "default", "query": "where Type = 'Warning'"}}
{"command": "logs", "flags": {"name": "my-pod", "namespace": "default", "tail": 100, "query": "where level = 'ERROR'"}}
```

## Common Pitfalls

| Mistake | Correct Form |
|---------|-------------|
| Missing quotes: `Name = my-pod` | `Name = 'my-pod'` |
| Parentheses for IN: `Status in ('Running')` | `Status in ['Running']` |
| Number as string: `Restarts = '5'` | `Restarts = 5` |
| Spaces in column name: `Last Seen` | `Last_Seen` |
| Both ORDER BY and SORT BY | Use one or the other, not both |

## Quick Reference Card

```
QUERY STRUCTURE
  [SELECT fields] [WHERE condition] [ORDER BY field [ASC|DESC]] [LIMIT n]

EXAMPLES
  Status = 'Running'                                  (bare expression, "where" auto-prepended)
  where Restarts > 0 order by Restarts desc limit 10
  fields.map is not null                              (nested field null check)
  select Name, Status where Restarts > 0 order by Name limit 5

DATA TYPES
  Strings       'single quoted'
  Numbers       42, 3.14
  SI Units      Ki  Mi  Gi  Ti  Pi
  Booleans      true, false
  Arrays        ['a', 'b', 'c']
  Null          null

COMPARISONS     =  !=  <>  <  <=  >  >=
ARITHMETIC      +  -  *  /  %
STRINGS         like  ilike  ~= (regex)  ~! (regex not)
LOGIC           and  or  not  ( )
SETS            in [...]  not in [...]  between X and Y
NULLS           is null  is not null

FUNCTIONS
  len(field)          array/string length
  any(cond)           true if any element matches
  all(cond)           true if all elements match
  sum(field)          sum of numeric array values

FIELD ACCESS
  field               simple column name
  field.sub           dot notation (nested, e.g. fields.map)
  field[0]            index access (zero-based)
  field[*].sub        wildcard (all elements)
  Last_Seen           underscore replaces spaces in column names

SHORTHAND
  "Name ~= 'nginx'"  is equivalent to  "where Name ~= 'nginx'"
  bare expressions without a keyword get "where" prepended automatically

SELECT BEHAVIOR
  table / markdown    ignored (original columns always shown)
  json / yaml         applied (only selected fields output)

SORTING         order by field [asc|desc]  (or: sort by)
LIMITING        limit N
```

## Further Reading

- [CLI Usage](cli-usage.md) -- command flags and examples
- [MCP Server](mcp-server.md) -- using queries via the MCP tools
- [TSL on GitHub](https://github.com/yaacov/tree-search-language) -- the TSL library
