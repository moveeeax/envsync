# envsync

[![ci](https://github.com/moveeeax/envsync/actions/workflows/ci.yml/badge.svg)](https://github.com/moveeeax/envsync/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Stop shipping broken configs. `envsync` validates a project's `.env` against a
schema-annotated `.env.example`, checking that every variable is present and
correctly typed before your app boots.

## Why

`.env.example` files drift. A new required variable lands, teammates forget to
add it locally, CI has no idea, and the app crashes at runtime with a cryptic
error. `envsync` turns the example file into an enforceable contract: annotate
each variable with its type and whether it is required, then fail fast.

## Install

```bash
go install github.com/moveeeax/envsync@latest
```

Or build from source:

```bash
git clone https://github.com/moveeeax/envsync
cd envsync && go build -o envsync .
```

## Usage

```bash
envsync check --example .env.example --env .env
```

Flags:

| Flag            | Default        | Description                                        |
| --------------- | -------------- | -------------------------------------------------- |
| `-e, --example` | `.env.example` | Schema-annotated example file                      |
| `-f, --env`     | `.env`         | The `.env` file to validate                        |
| `--json`        | `false`        | Emit machine-readable JSON                         |
| `--strict`      | `false`        | Treat undeclared (extra) variables as failures     |
| `--fix`         | `false`        | Scaffold missing keys into `.env`, then check      |
| `--show-values` | `false`        | Echo the offending value — **may print secrets**   |

The command exits non-zero when validation fails, so it drops straight into CI
or a pre-commit hook.

## Schema annotations

Annotations live in comments in `.env.example` and apply to the variable on the
next line (or inline after the value):

```dotenv
# @type=int @required
PORT=8080

# @type=enum(dev,staging,prod) @required
APP_ENV=dev

# @type=url @required
DATABASE_URL=postgres://localhost:5432/app

# @type=bool
DEBUG=false

TIMEOUT=30 # @type=int
```

Supported types: `int`, `bool`, `url`, `enum(a,b,c)`, and `string` (the
default). Add `@required` to demand a present, non-empty value.

Enum members may also be declared with the standalone `@enum=` form, and spaces
after the commas are fine in either form:

```dotenv
# @enum=dev, staging, prod @required
APP_ENV=dev
```

## What it reports

- **missing** — a `@required` variable is absent, or present but empty, in `.env`
- **mismatch** — a value does not satisfy its declared type/enum
- **extra** — a variable exists in `.env` but is not declared in the example
  (reported always; only fails the run under `--strict`)

### Example

```console
$ envsync check -e examples/.env.example -f broken.env
mismatch  PORT: expected int, got a 3-character value
mismatch  APP_ENV: expected one of [dev, staging, prod], got a 5-character value
missing   DATABASE_URL: required variable is missing

3 issue(s) found
$ echo $?
1
```

### Values are never printed

`envsync` is built to run in CI, where stdout is archived and often public. A
variable that fails its type check is very frequently a credential — a token
pasted where an `int` was expected, a password that should have been a URL — so
the reported value is reduced to its length. The list of *allowed* enum values
is still shown, because that comes from the committed `.env.example`, not from
your `.env`.

Pass `--show-values` when you are debugging locally and want the raw value:

```console
$ envsync check --show-values
mismatch  PORT: expected int, got "xyz"
```

JSON output for pipelines:

```bash
envsync check --json | jq '.issues[] | select(.kind=="missing")'
```

## Scaffold a new .env

```bash
envsync check --fix
```

Creates `.env` from the example (or appends only the keys you are missing —
existing values are never overwritten), then validates the result.

The file is written by staging the new content alongside it and renaming it into
place, so an interrupted or failing run cannot leave a half-written `.env`
behind. The result is always mode `0600`, which also tightens a `.env` that was
previously left group- or world-readable.

## CI

`envsync` is a single static binary and a natural CI gate:

```yaml
- run: go install github.com/moveeeax/envsync@latest
- run: envsync check --example .env.example --env .env --strict
```

## License

[MIT](LICENSE)
