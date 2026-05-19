# utils-time-object (delta)

## MODIFIED Requirements

### Requirement: Pseudo-random int in range

`DateTime` SHALL provide `Random(max int64) int64` that returns a value in `[0, max)` using a package-level reused `*rand.Rand` protected by a mutex. The implementation SHALL re-seed with `time.Now().UnixMilli()` on each call to preserve the millisecond factor behavior.

#### Scenario: Valid max

- **WHEN** `max` is a positive integer and `Random` is invoked
- **THEN** the result SHALL satisfy `0 <= result < max`

#### Scenario: Non-positive max

- **WHEN** `max <= 0`
- **THEN** `Random(max)` SHALL return `0`

#### Scenario: Reused generator

- **WHEN** `Random` is invoked multiple times in succession
- **THEN** the implementation SHALL NOT allocate a new `rand.Rand` per call
