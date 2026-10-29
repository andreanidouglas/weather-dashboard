# Agent instructions

Apply the relevant skill before making changes or reviewing code in its area. If a skill is unavailable, follow the guard below directly and do not claim to have used the skill.

## Code-style guard

- For Go code changes or reviews, use the `golang-code-style` skill. Keep code idiomatic and consistent with nearby code; favor clarity, focused functions, and simple control flow over cleverness.
- Run `gofmt` on changed Go files. Do not introduce a formatter or style rule that conflicts with the project.
- For non-Go code, follow the conventions and tooling already used in the affected area.

## Security guard

- For work involving authentication, configuration or secrets, HTTP handlers or middleware, outbound requests, templates, dependencies, or deployment files, use the `golang-security` skill before editing or reviewing. Also use it when a change introduces or alters a trust boundary or handles untrusted input.
- Never hard-code, print, commit, or include real API keys or other secrets in examples, logs, tests, or fixtures. Use fake credentials in tests.
- Treat request data and upstream API responses as untrusted. Validate and bound user-controlled inputs; do not let them select arbitrary outbound URLs. Preserve `templ`'s escaping and avoid rendering user or upstream content as raw HTML.
- Keep error responses free of secrets and internal details. Avoid logging API keys or full sensitive request data.
- Review dependency and container/deployment changes for unnecessary privileges, exposed secrets, and unintended network or filesystem access.

## Testing guard

- For behavior changes, use the `golang-testing` skill when adding or updating Go tests. Add focused tests for changed behavior; prefer the existing `httptest`-based mocks. Tests must not require a live OpenWeatherMap API key or make external network calls.
- Run `go test ./...` for Go changes. Run `go test -race ./...` when changing shared state, caching, or concurrency, and `go vet ./...` when practical.
- For changes to `*.templ`, regenerate the corresponding generated Go files with `templ generate`; do not edit generated `*_templ.go` files by hand.
- The frontend currently has no test script. For frontend or CSS changes, run the relevant available build/check and report clearly if there is no automated test for it.
- Avoid using `make` solely as a test command: the `server` target runs `go mod tidy` and may modify dependency files.
