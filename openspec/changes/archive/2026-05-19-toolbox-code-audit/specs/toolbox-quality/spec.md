# toolbox-quality

## ADDED Requirements

### Requirement: JSON helpers decode into caller pointer correctly

`utils_json.JSONDecode` SHALL unmarshal into the `result` argument without an extra indirection layer, matching `encoding/json.Unmarshal` contract.

#### Scenario: Decode into struct pointer

- **WHEN** caller passes a pointer to an empty struct and valid JSON object string
- **THEN** the struct fields SHALL be populated after `JSONDecode` returns nil error

### Requirement: JSON encode exposes errors

The `utils_json` package SHALL provide `JSONEncodeE` returning `(string, error)` on marshal failure.

#### Scenario: Marshal failure

- **WHEN** value cannot be marshaled to JSON
- **THEN** `JSONEncodeE` SHALL return non-nil error

### Requirement: Safe random int uses correct upper bound

`utils_random.IntWithSafety` SHALL use a cryptographically secure source with an upper bound of 2^63 (implemented as `1 << 63` in Go), not the bitwise XOR of literals `2` and `63`.

#### Scenario: Result within int64 range

- **WHEN** `IntWithSafety` is invoked successfully
- **THEN** the result SHALL be non-negative and less than `1 << 63`

### Requirement: RandUtil Intn guards non-positive n

`RandUtil.Intn` SHALL not panic when `n <= 0`; it SHALL return `0` for non-positive `n`.

#### Scenario: Zero or negative n

- **WHEN** `Intn(0)` or `Intn(-1)` is called
- **THEN** the method SHALL return `0` without panicking

### Requirement: CSV library code does not panic on I/O errors

`utils_csv.OnceFullWrite` SHALL return errors from file open and row write operations and SHALL NOT call `panic` or `log.Fatal`.

#### Scenario: Write failure

- **WHEN** writing a record fails
- **THEN** `OnceFullWrite` SHALL return a non-nil error

### Requirement: ExecCmd respects context cancellation

`utils_exec.ExecCmd` SHALL use context-aware command execution so cancellation terminates or prevents completion with context error.

#### Scenario: Cancelled context

- **WHEN** context is cancelled before command completes
- **THEN** `ExecCmd` result SHALL report context cancellation error

### Requirement: Documented change log for audit

The repository SHALL include `CHANGELOG.md` listing audit fixes by package with date and **BREAKING** notes where applicable.

#### Scenario: Changelog present

- **WHEN** audit change is applied
- **THEN** `CHANGELOG.md` SHALL contain entries for fixed issues listed in the change proposal
