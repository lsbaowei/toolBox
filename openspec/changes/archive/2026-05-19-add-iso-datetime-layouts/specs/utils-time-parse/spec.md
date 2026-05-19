# utils-time-parse (delta)

## ADDED Requirements

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
