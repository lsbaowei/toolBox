# utils-time-object Specification

## Purpose
TBD - created by archiving change add-utils-time-object. Update Purpose after archive.
## Requirements
### Requirement: Construct DateTime with optional base time

The package SHALL provide `New(base *time.Time) *DateTime`. When `base` is nil, the object SHALL use the current instant from `time.Now()` in the local process environment.

#### Scenario: New with nil base

- **WHEN** caller invokes `New(nil)`
- **THEN** the object's underlying instant SHALL be approximately the current time

#### Scenario: New with explicit base

- **WHEN** caller invokes `New(&t)` for a specific `time.Time` `t`
- **THEN** `Time()` SHALL return an instant equal to `t`

### Requirement: Expose base time and formatting

`DateTime` SHALL expose the base instant via `Time() time.Time` and format it via `Format(layout string) string` equivalent to `Time().Format(layout)`.

#### Scenario: Format with layout

- **WHEN** caller invokes `Format("2006-01-02")` on a known base time
- **THEN** the returned string SHALL match `Time().Format("2006-01-02")`

#### Scenario: Common format shortcuts

- **WHEN** caller invokes documented shortcut format methods (e.g. RFC3339, date-only, date-time)
- **THEN** each SHALL return the same result as `Format` with the corresponding layout constant or string

### Requirement: Duration and calendar arithmetic return new objects

`DateTime` SHALL provide `Add(d time.Duration) *DateTime` and `AddDate(years, months, days int) *DateTime` that delegate to `time.Time` and return a new `DateTime` without mutating the receiver.

#### Scenario: Add duration

- **WHEN** receiver has base time `t` and caller invokes `Add(d)`
- **THEN** the new object's `Time()` SHALL equal `t.Add(d)`

#### Scenario: AddDate

- **WHEN** caller invokes `AddDate(0, 1, 0)` on a known date
- **THEN** the new object's calendar components SHALL reflect one month added in the object's location

### Requirement: Day boundaries in base location

`DateTime` SHALL provide `StartOfDay` and `EndOfDay` returning new objects at `00:00:00` and `23:59:59.999999999` on the same calendar date as the base time in the base location.

#### Scenario: Start of day

- **WHEN** base time is `2024-06-15 14:30:00` in a fixed location
- **THEN** `StartOfDay().Time()` SHALL be `2024-06-15 00:00:00` in that location

#### Scenario: End of day

- **WHEN** base time is `2024-06-15 14:30:00` in a fixed location
- **THEN** `EndOfDay().Time()` SHALL be `2024-06-15 23:59:59.999999999` in that location

### Requirement: Week boundaries Monday through Sunday

`DateTime` SHALL provide `StartOfWeek` and `EndOfWeek` where the week starts Monday `00:00:00` and ends Sunday `23:59:59` in the base location.

#### Scenario: Mid-week base

- **WHEN** base time is Wednesday `2024-06-12 10:00:00`
- **THEN** `StartOfWeek()` SHALL be Monday `2024-06-10 00:00:00` and `EndOfWeek()` SHALL be Sunday `2024-06-16 23:59:59` in the same location

### Requirement: Month boundaries

`DateTime` SHALL provide `StartOfMonth` and `EndOfMonth` at the first day `00:00:00` and last day `23:59:59` of the month containing the base time in the base location.

#### Scenario: Mid-month base

- **WHEN** base time is `2024-06-15 12:00:00`
- **THEN** `StartOfMonth()` SHALL be `2024-06-01 00:00:00` and `EndOfMonth()` SHALL be `2024-06-30 23:59:59`

#### Scenario: February non-leap year

- **WHEN** base time is in February 2023
- **THEN** `EndOfMonth()` SHALL be `2023-02-28 23:59:59`

### Requirement: Remaining seconds until base time

`DateTime` SHALL provide `RemainingSeconds(other *time.Time) int64` that returns the base instant's Unix seconds minus the reference instant's Unix seconds. When `other` is nil, the reference SHALL be the current time at call.

#### Scenario: Remaining seconds from now

- **WHEN** base time is 1000 seconds after epoch and `other` is nil while current time is 400 seconds after epoch
- **THEN** `RemainingSeconds(nil)` SHALL return `600`

#### Scenario: Remaining seconds versus explicit other

- **WHEN** base Unix is `2000` and `other` is `1500` seconds after epoch
- **THEN** `RemainingSeconds(&other)` SHALL return `500`

#### Scenario: Negative when other is after base

- **WHEN** base Unix is `1000` and `other` is `1200` seconds after epoch
- **THEN** `RemainingSeconds(&other)` SHALL return `-200`

### Requirement: Pseudo-random int in range

`DateTime` SHALL provide `Random(max int64) int64` that returns a value in `[0, max)` using `time.Now().UnixMilli()` as the random seed factor at invocation time.

#### Scenario: Valid max

- **WHEN** `max` is a positive integer and `Random` is invoked
- **THEN** the result SHALL satisfy `0 <= result < max`

#### Scenario: Non-positive max

- **WHEN** `max <= 0`
- **THEN** `Random(max)` SHALL return `0`

