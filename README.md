# quail-core

Shared Go library extracted from
[`quail-review`](https://github.com/spriteCloud/quail-review) so
[`quail-platform`](https://github.com/spriteCloud/quail-platform) (and
quail-review itself) can depend on a single source of truth for the
underlying machinery — AST extraction, web probing, template
generation, planning, LLM client, schema parsers.

## Packages

`ast`, `asyncapi`, `compat`, `composer`, `config`, `diff`, `gen`, `gh`,
`graphql`, `grpc`, `heal`, `integration`, `ledger`, `llm`, `log`,
`merge`, `mindmap`, `openapi`, `plan`, `probe`, `prompt`.

## Usage

```go
import "github.com/spriteCloud/quail-core/probe"
import "github.com/spriteCloud/quail-core/llm"
```

## Consumers

- [quail-review](https://github.com/spriteCloud/quail-review) — CLI binary
- [quail-platform](https://github.com/spriteCloud/quail-platform) — multi-tenant webapp

## License

See [`LICENSE`](./LICENSE).
