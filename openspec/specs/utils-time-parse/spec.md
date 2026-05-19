# utils-time-parse Specification

## Purpose
TBD - created by archiving change improve-utils-time-parse. Update Purpose after archive.
## Requirements
### Requirement: Parse Unix timestamp in safe range

The package SHALL parse decimal integer strings as Unix timestamps, distinguishing seconds from milliseconds using the same rules as `ParseUnixTimestamp` (`secThreshold` and `msThreshold`).

#### Scenario: Seconds in safe range

- **WHEN** input is a decimal string whose numeric value is greater than 0 and less than `secThreshold`
- **THEN** the result SHALL represent that instant using second resolution

#### Scenario: Milliseconds in safe range

- **WHEN** input is a decimal string whose numeric value is greater than or equal to `secThreshold` and less than `msThreshold`
- **THEN** the result SHALL represent that instant using millisecond resolution

#### Scenario: Out of safe range

- **WHEN** input is a decimal string outside the supported second/millisecond ranges (including non-positive values if not supported)
- **THEN** parsing SHALL return a non-nil error

### Requirement: ParseTimeUTC returns UTC semantics

`ParseTimeUTC` SHALL parse supported date/time layout strings and return a `time.Time` whose instant is correct for the input; for layouts without timezone information, wall-clock components SHALL be interpreted in UTC.

#### Scenario: RFC3339 with offset

- **WHEN** input is an RFC3339 string with a numeric offset (e.g. `2025-01-02T15:04:05+08:00`)
- **THEN** the returned instant SHALL match the specified offset

#### Scenario: Date without timezone

- **WHEN** input matches a configured layout without timezone (e.g. `2024-05-28` or `2024-05-28 12:34:56`)
- **THEN** the returned time SHALL use UTC as the location for wall-clock fields

#### Scenario: Unix string via ParseTimeUTC

- **WHEN** input is a valid Unix second or millisecond string per safe-range rules
- **THEN** the returned time SHALL equal the UTC instant from `ParseUnixTimestamp` converted with `.UTC()`

### Requirement: ParseTime applies default timezone

`ParseTime` SHALL accept the same string formats as `ParseTimeUTC` for non-integer inputs, and SHALL apply `defaultTZ` to layouts without timezone information.

#### Scenario: Layout without zone uses defaultTZ

- **WHEN** input matches a layout without timezone information and `defaultTZ` is provided
- **THEN** the returned time SHALL represent the same wall-clock fields in `defaultTZ`

#### Scenario: Layout with zone ignores defaultTZ for instant

- **WHEN** input includes explicit timezone information in the string
- **THEN** the returned instant SHALL follow the string's zone/offset, not reinterpret wall clock in `defaultTZ`

#### Scenario: Unix string via ParseTime

- **WHEN** input is a valid Unix second or millisecond string per safe-range rules
- **THEN** the returned time SHALL equal the instant from `ParseUnixTimestamp` with location set to `defaultTZ` via `.In(defaultTZ)`

### Requirement: Unsupported format fails clearly

When input is neither a valid integer timestamp nor parseable by any configured layout, parsing functions SHALL return a non-nil error and a zero `time.Time`.

#### Scenario: Garbage input

- **WHEN** input is a non-numeric string that does not match any supported layout
- **THEN** parsing SHALL return an error

### Requirement: Regression tests cover documented formats

The `utils_time` package tests SHALL include table-driven cases for documented Unix, RFC, and custom layouts used in production, and SHALL fail on intentional behavior changes.

#### Scenario: Test suite passes

- **WHEN** `go test ./utils_time/...` is run after changes
- **THEN** all tests SHALL pass without skipping core scenarios listed in the change tasks

### Requirement: Parse ISO datetime without timezone suffix

The package SHALL parse `2006-01-02T15:04:05` strings using wall-clock semantics in UTC (`ParseTimeUTC`) or `defaultTZ` (`ParseTime`).

#### Scenario: UTC wall

- **WHEN** input is `2025-01-02T15:04:05` and `ParseTimeUTC` is called
- **THEN** the result SHALL be 2025-01-02 15:04:05 UTC

#### Scenario: Location wall

- **WHEN** input is `2025-01-02T15:04:05` and `ParseTime` is called with a fixed location
- **THEN** wall-clock fields SHALL match 2025-01-02 15:04:05 in that location

### Requirement: Parse task 5 ISO samples with offset and fractional seconds

The package SHALL successfully parse README task 5 samples including nanosecond and millisecond fractions with `+08:00` offset.

#### Scenario: Nanoseconds and offset

- **WHEN** input is `2025-11-19T11:33:19.920584349+08:00`
- **THEN** parsing SHALL succeed

#### Scenario: Milliseconds and offset

- **WHEN** input is `2026-05-18T15:29:04.527+08:00`
- **THEN** parsing SHALL succeed

#### Scenario: Offset without fractional seconds

- **WHEN** input is `2025-01-02T15:04:05+08:00`
- **THEN** parsing SHALL succeed

