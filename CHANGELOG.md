# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security

- Mismatch messages no longer echo the variable's value, in either human or
  JSON output. A credential pasted into the wrong variable used to end up in CI
  logs. Only the value's length is reported now; pass `--show-values` to opt
  back in for local debugging (#1).
- `check --fix` writes `.env` through a temporary file and an atomic rename,
  and the result is always mode `0600`. Previously it appended in place, so a
  failed write could leave a half-written line, and a world-readable `.env`
  stayed world-readable after credentials were added to it (#1).

### Fixed

- `# @enum=dev, staging, prod` with spaces after the commas now accepts every
  member instead of only the first one (#1).
- A required variable that is present but empty is reported as `missing`
  rather than `mismatch`, matching the README and the `jq` recipe (#1).
- A value that starts with `#` directly after `=` (for example
  `THEME_COLOR=#336699`) is kept instead of being parsed as an empty value
  with a comment (#2).
- `check --fix` re-quotes example defaults that need quoting (an embedded
  ` #` or leading/trailing whitespace), so they survive the round trip (#2).

### Changed

- Go 1.26 or newer is required to build.
- `github.com/spf13/cobra` updated from v1.8.0 to v1.10.2.

## [0.1.0] - 2026-07-20

First release. This version prints offending values, which may include
secrets, into CI logs; upgrade to the next release.

[Unreleased]: https://github.com/moveeeax/envsync/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/moveeeax/envsync/releases/tag/v0.1.0
