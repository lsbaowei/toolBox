## 1. Package Setup

- [x] 1.1 Create a new Go package directory for gradual cache version release utilities.
- [x] 1.2 Define public `Config`, `Manager`, and result types for selecting cache versions.
- [x] 1.3 Add validation for release duration, business identifier, and version input.

## 2. Core Release State

- [x] 2.1 Implement stable version initialization when the first third-party version is observed.
- [x] 2.2 Implement release stage state with base version, target version, start time, and release duration.
- [x] 2.3 Implement active target updates when newer versions are observed during an active release.
- [x] 2.4 Complete release stages and promote the latest active target version to the stable version.

## 3. Gradual Selection Logic

- [x] 3.1 Implement deterministic bucket calculation from business identifier and target version.
- [x] 3.2 Calculate release progress from an injected or explicit current time.
- [x] 3.3 Return the previous stable version or active target version according to progress threshold.
- [x] 3.4 Persist sticky release decisions for identifiers that have received the active target version.
- [x] 3.5 Clean up sticky state for completed release stages.

## 4. Tests

- [x] 4.1 Add tests for first-version initialization and stable version selection.
- [x] 4.2 Add tests for gradual release progress at start, partial progress, and completion.
- [x] 4.3 Add tests proving repeated selection for the same identifier is deterministic.
- [x] 4.4 Add tests proving sticky identifiers do not fall back during an active release.
- [x] 4.5 Add tests for multiple versions arriving during an active release and updating the target to the latest version.
- [x] 4.6 Add tests for invalid release duration and empty business identifier errors.
- [x] 4.7 Add tests using controlled time to verify deterministic progress calculation.

## 5. Documentation

- [x] 5.1 Update `README.md` with the new package purpose and a minimal usage example.
- [x] 5.2 Document business identifier expectations and the single-process state limitation.
