## ADDED Requirements

### Requirement: Select cache version by release progress

The `gradual-cache-version-release` capability SHALL provide an API that accepts a stable business identifier, the third-party current version, and release configuration, then returns the cache version that the caller MUST use for the current request.

#### Scenario: First version becomes stable

- **WHEN** the selector receives a version and no previous stable version exists
- **THEN** the selector SHALL return that version as the stable cache version without starting a gradual release

#### Scenario: New version starts gradual release

- **WHEN** the selector has a stable version and receives a different third-party version
- **THEN** the selector SHALL start a release stage for the new version and return either the stable version or the new version according to release progress

#### Scenario: Release window completes

- **WHEN** the configured release duration has fully elapsed for the active target version
- **THEN** the selector SHALL return the target version for all business identifiers and mark it as the stable version

### Requirement: Release traffic gradually during configured duration

The capability SHALL increase the proportion of business identifiers released to the target version over the configured release duration, rather than switching all identifiers immediately.

#### Scenario: Release starts at zero progress

- **WHEN** a new release stage has just started
- **THEN** the selector SHALL keep unreleased business identifiers on the previous stable version

#### Scenario: Release advances before completion

- **WHEN** the active release stage is partially elapsed
- **THEN** the selector SHALL release only the subset of business identifiers whose deterministic bucket is within the current progress threshold

#### Scenario: Same identifier is deterministic

- **WHEN** the same business identifier is evaluated repeatedly during the same release stage and state has not changed
- **THEN** the selector SHALL return the same cache version each time

### Requirement: Keep sticky release decisions

The capability SHALL preserve sticky release decisions so that a business identifier that has been released to a newer version does not fall back to an older version in later requests.

#### Scenario: Identifier already released

- **WHEN** a business identifier has previously received the active target version
- **THEN** later selections for that identifier SHALL return at least that target version while the release stage remains active

#### Scenario: Release stage completes

- **WHEN** an active release stage completes
- **THEN** sticky records for that completed stage SHALL no longer be required to keep identifiers on the completed version

### Requirement: Use latest discovered version during active release

The capability SHALL use the latest discovered third-party version as the active release target when versions appear faster than the configured release duration can complete.

#### Scenario: Newer version arrives during active release

- **WHEN** the selector is releasing `v1.1.1` and receives `v1.1.2`, `v1.1.3`, and `v1.1.4` before the active release completes
- **THEN** the selector SHALL update the active release target to `v1.1.4` and ignore the intermediate versions as standalone release targets

#### Scenario: Release cadence is preserved

- **WHEN** the active release target is updated to a newer version
- **THEN** the selector SHALL preserve the current release stage timing instead of restarting the release duration

#### Scenario: Eventually reaches latest discovered version

- **WHEN** the active release stage has completed
- **THEN** the selector SHALL return the latest discovered version for all business identifiers

### Requirement: Validate release configuration

The capability SHALL validate release configuration before selecting versions, including release duration and business identifier input.

#### Scenario: Invalid release duration

- **WHEN** the release duration is zero or negative
- **THEN** the selector SHALL return an error instead of performing a gradual release calculation

#### Scenario: Empty business identifier

- **WHEN** the selector is called with an empty business identifier after a stable version already exists
- **THEN** the selector SHALL return an error because deterministic and sticky release decisions require a stable identifier

### Requirement: Support testable time behavior

The capability SHALL support an injectable time source or explicit current time so release progress can be tested deterministically.

#### Scenario: Caller controls current time

- **WHEN** tests evaluate the same release stage at different controlled times
- **THEN** the selector SHALL calculate release progress from the supplied time rather than from hidden wall-clock state
