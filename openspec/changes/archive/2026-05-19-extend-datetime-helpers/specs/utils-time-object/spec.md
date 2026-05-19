# utils-time-object (delta)

## ADDED Requirements

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
