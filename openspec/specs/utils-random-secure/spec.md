# utils-random-secure Specification

## Purpose
TBD - created by archiving change optimize-rand-reuse. Update Purpose after archive.
## Requirements
### Requirement: Secure random int in range

The `utils_random` package SHALL provide `SecureIntn(max int64) (int64, error)` that returns a uniform value in `[0, max)` using `crypto/rand`.

#### Scenario: Valid max

- **WHEN** `max` is positive and `SecureIntn` succeeds
- **THEN** the result SHALL satisfy `0 <= result < max`

#### Scenario: Non-positive max

- **WHEN** `max <= 0`
- **THEN** `SecureIntn` SHALL return `0` and nil error

### Requirement: Secure random int64 full range

The package SHALL provide `SecureInt64() (int64, error)` returning a value in `[0, 2^63)` via `crypto/rand`, equivalent to the corrected `IntWithSafety` semantics.

#### Scenario: Successful draw

- **WHEN** `SecureInt64` is called and randomness is available
- **THEN** it SHALL return a non-negative int64 and nil error

### Requirement: Document secure versus pseudo random

README task 4 SHALL document when to use `SecureIntn` / `SecureInt64` versus `RandUtil` / `DateTime.Random`.

#### Scenario: Documentation present

- **WHEN** the change is applied
- **THEN** README or example test SHALL show at least one secure and one pseudo-random usage snippet

