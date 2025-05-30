# Experimental: Do If rules (logs content matching rules)

This is experimental feature and represents an advanced version of `match_fields`.
The Do If rules are a tree of nodes. The tree is stored in the Do If Checker instance.
When Do If Checker's Match func is called it calls to the root Match func and then
the chain of Match func calls are performed across the whole tree.

## Node types
**`FieldOp`** Type of node where matching rules for fields are stored.

<br>

**`LengthCmpOp`** Type of node where matching rules for byte length and array length are stored.

<br>

**`TimestampCmpOp`** Type of node where matching rules for timestamps are stored.

<br>

**`CheckTypeOp`** Type of node where matching rules for check types are stored.

<br>

**`LogicalOp`** Type of node where logical rules for applying other rules are stored.

<br>


## Field op node
DoIf field op node is considered to always be a leaf in the DoIf tree. It checks byte representation of the value by the given field path.
Array and object values are considered as not matched since encoding them to bytes leads towards large CPU and memory consumption.

Params:
  - `op` - value from field operations list. Required.
  - `field` - path to field in JSON tree. If empty, root value is checked. Path to nested fields is delimited by dots `"."`, e.g. `"field.subfield"` for `{"field": {"subfield": "val"}}`.
  If the field name contains dots in it they should be shielded with `"\"`, e.g. `"exception\.type"` for `{"exception.type": "example"}`. Default empty.
  - `values` - list of values to check field. Required non-empty.
  - `case_sensitive` - flag indicating whether checks are performed in case sensitive way. Default `true`.
    Note: case insensitive checks can cause CPU and memory overhead since every field value will be converted to lower letters.

Example:
```yaml
pipelines:
  tests:
    actions:
      - type: discard
        do_if:
          op: suffix
          field: pod
          values: [pod-1, pod-2]
          case_sensitive: true
```


## Field operations
**`Equal`** checks whether the field value is equal to one of the elements in the values list.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: equal
          field: pod
          values: [test-pod-1, test-pod-2]
```

result:
```
{"pod":"test-pod-1","service":"test-service"}   # discarded
{"pod":"test-pod-2","service":"test-service-2"} # discarded
{"pod":"test-pod","service":"test-service"}     # not discarded
{"pod":"test-pod","service":"test-service-1"}   # not discarded
```

<br>

**`Contains`** checks whether the field value contains one of the elements the in values list.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: contains
          field: pod
          values: [my-pod, my-test]
```

result:
```
{"pod":"test-my-pod-1","service":"test-service"}     # discarded
{"pod":"test-not-my-pod","service":"test-service-2"} # discarded
{"pod":"my-test-pod","service":"test-service"}       # discarded
{"pod":"test-pod","service":"test-service-1"}        # not discarded
```

<br>

**`Prefix`** checks whether the field value has prefix equal to one of the elements in the values list.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: prefix
          field: pod
          values: [test-1, test-2]
```

result:
```
{"pod":"test-1-pod-1","service":"test-service"}   # discarded
{"pod":"test-2-pod-2","service":"test-service-2"} # discarded
{"pod":"test-pod","service":"test-service"}       # not discarded
{"pod":"test-pod","service":"test-service-1"}     # not discarded
```

<br>

**`Suffix`** checks whether the field value has suffix equal to one of the elements in the values list.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: suffix
          field: pod
          values: [pod-1, pod-2]
```

result:
```
{"pod":"test-1-pod-1","service":"test-service"}   # discarded
{"pod":"test-2-pod-2","service":"test-service-2"} # discarded
{"pod":"test-pod","service":"test-service"}       # not discarded
{"pod":"test-pod","service":"test-service-1"}     # not discarded
```

<br>

**`Regex`** checks whether the field matches any regex from the values list.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: regex
          field: pod
          values: [pod-\d, my-test.*]
```

result:
```
{"pod":"test-1-pod-1","service":"test-service"}       # discarded
{"pod":"test-2-pod-2","service":"test-service-2"}     # discarded
{"pod":"test-pod","service":"test-service"}           # not discarded
{"pod":"my-test-pod","service":"test-service-1"}      # discarded
{"pod":"my-test-instance","service":"test-service-1"} # discarded
{"pod":"service123","service":"test-service-1"}       # not discarded
```

<br>


## Logical op node
DoIf logical op node is a node considered to be the root or an edge between nodes.
It always has at least one operand which are other nodes and calls their checks
to apply logical operation on their results.

Params:
  - `op` - value from logical operations list. Required.
  - `operands` - list of another do-if nodes. Required non-empty.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: and
          operands:
            - op: equal
              field: pod
              values: [test-pod-1, test-pod-2]
              case_sensitive: true
            - op: equal
              field: service
              values: [test-service]
              case_sensitive: true
```


## Logical operations
**`Or`** accepts at least one operand and returns true on the first returned true from its operands.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: or
          operands:
            - op: equal
              field: pod
              values: [test-pod-1, test-pod-2]
            - op: equal
              field: service
              values: [test-service]
```

result:
```
{"pod":"test-pod-1","service":"test-service"}   # discarded
{"pod":"test-pod-2","service":"test-service-2"} # discarded
{"pod":"test-pod","service":"test-service"}     # discarded
{"pod":"test-pod","service":"test-service-1"}   # not discarded
```

<br>

**`And`** accepts at least one operand and returns true if all operands return true
(in other words returns false on the first returned false from its operands).

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: and
          operands:
            - op: equal
              field: pod
              values: [test-pod-1, test-pod-2]
            - op: equal
              field: service
              values: [test-service]
```

result:
```
{"pod":"test-pod-1","service":"test-service"}   # discarded
{"pod":"test-pod-2","service":"test-service-2"} # not discarded
{"pod":"test-pod","service":"test-service"}     # not discarded
{"pod":"test-pod","service":"test-service-1"}   # not discarded
```

<br>

**`Not`** accepts exactly one operand and returns inverted result of its operand.

Example:
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: not
          operands:
            - op: equal
              field: service
              values: [test-service]
```

result:
```
{"pod":"test-pod-1","service":"test-service"}   # not discarded
{"pod":"test-pod-2","service":"test-service-2"} # discarded
{"pod":"test-pod","service":"test-service"}     # not discarded
{"pod":"test-pod","service":"test-service-1"}   # discarded
```

<br>


## Length comparison op node
DoIf length comparison op node is considered to always be a leaf in the DoIf tree like DoIf field op node.
It contains operation that compares field length in bytes or array length (for array fields) with certain value.

Params:
  - `op` - must be `byte_len_cmp` or `array_len_cmp`. Required.
  - `field` - name of the field to apply operation. Required.
  - `cmp_op` - comparison operation name (see below). Required.
  - `value` - integer value to compare length with. Required non-negative.

Example 1 (byte length comparison):
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: byte_len_cmp
          field: pod_id
          cmp_op: lt
          value: 5
```

Result:
```
{"pod_id":""}      # discarded
{"pod_id":123}     # discarded
{"pod_id":12345}   # not discarded
{"pod_id":123456}  # not discarded
```

Example 2 (array length comparison):

```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: array_len_cmp
          field: items
          cmp_op: lt
          value: 2
```

Result:
```
{"items":[]}         # discarded
{"items":[1]}        # discarded
{"items":[1, 2]}     # not discarded
{"items":[1, 2, 3]}  # not discarded
{"items":"1"}        # not discarded ('items' is not an array)
{"numbers":[1]}      # not discarded ('items' not found)
```

Possible values of field `cmp_op`: `lt`, `le`, `gt`, `ge`, `eq`, `ne`.
They denote corresponding comparison operations.

| Name | Op |
|------|----|
| `lt` | `<` |
| `le` | `<=` |
| `gt` | `>` |
| `ge` | `>=` |
| `eq` | `==` |
| `ne` | `!=` |

## Timestamp comparison op node
DoIf timestamp comparison op node is considered to always be a leaf in the DoIf tree like DoIf field op node.
It contains operation that compares timestamps with certain value.

Params:
  - `op` - must be `ts_cmp`. Required.
  - `field` - name of the field to apply operation. Required. Field will be parsed with `time.Parse` function.
  - `format` - format for timestamps representation.
Can be specified as a datetime layout in Go [time.Parse](https://pkg.go.dev/time#Parse) format or by alias.
List of available datetime format aliases can be found [here](/pipeline/README.md#datetime-parse-formats).
Optional; default = `rfc3339nano`.
  - `cmp_op` - comparison operation name (same as for length comparison operations). Required.
  - `value` - timestamp value to compare field timestamps with. It must have `RFC3339Nano` format. Required.
Also, it may be `now` or `file_d_start`. If it is `now` then value to compare timestamps with is periodically updated current time.
If it is `file_d_start` then value to compare timestamps with will be program start moment.
  - `value_shift` - duration that adds to `value` before comparison. It can be negative. Useful when `value` is `now`.
Optional; default = 0.
  - `update_interval` - if `value` is `now` then you can set update interval for that value. Optional; default = 10s.
Actual cmp value in that case is `now + value_shift + update_interval`.

Example (discard all events with `timestamp` field value LESS than `2010-01-01T00:00:00Z`):
```yaml
pipelines:
  test:
    actions:
      - type: discard
        do_if:
          op: ts_cmp
          field: timestamp
          cmp_op: lt
          value: 2010-01-01T00:00:00Z
          format: 2006-01-02T15:04:05.999999999Z07:00
```

Result:
```
{"timestamp":"2000-01-01T00:00:00Z"}         # discarded
{"timestamp":"2008-01-01T00:00:00Z","id":1}  # discarded

{"pod_id":"some"}    # not discarded (no field `timestamp`)
{"timestamp":123}    # not discarded (field `timestamp` is not string)
{"timestamp":"qwe"}  # not discarded (field `timestamp` is not parsable)

{"timestamp":"2011-01-01T00:00:00Z"}  # not discarded (condition is not met)
```

## Check type op node
DoIf check type op node checks whether the type of the field node is the one from the list.

Params:
  - `op` - for that node the value is `check_type`
  - `field` - path to JSON node, can be empty string meaning root node, can be nested field `field.subfield` (if the field consists of `.` in the name, it must be shielded e.g. `exception\.type`)
  - `values` - list of types to check against. Allowed values are `object` (or `obj`), `array` (or `arr`), `number` (or `num`, matches both ints or floats), `string` (or `str`), `null`, `nil` (for the abscent fields). Values are deduplicated with respect to aliases on initialization to prevent redundant checks.

Example:

```yaml
- type: discard
  do_if:
    op: not
    operands:
      - op: check_type
        field: log
        values: [obj, arr]
```

result:
```
{"log":{"message":"test"}}   # not discarded
{"log":[{"message":"test"}]} # not discarded
{"log":"test"}               # discarded
{"log":123}                  # discarded
{"log":null}                 # discarded
{"not_log":{"test":"test"}}  # discarded
```

<br>*Generated using [__insane-doc__](https://github.com/vitkovskii/insane-doc)*