# envsync

> Stop shipping broken configs — validate .env against a typed .env.example.

**Status:** 🚧 In development

## Overview

Validate a project's .env against a schema-annotated .env.example, checking presence and types.

## Features

- Parse `.env` and `.env.example` including comment-based type/required annotations
- Report missing, extra and type-mismatched variables
- Support types (int, bool, url, enum) and required markers via annotations
- CI-friendly non-zero exit and JSON output
- Optional `--fix` to scaffold a `.env` from the example

## Stack

Go 1.22, dotenv parsing, `cobra`.

## Usage

```bash
envsync check --example .env.example --env .env
```

## License

MIT
