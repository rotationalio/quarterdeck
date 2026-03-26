# Zerolog → rlog: agent guide

Replace **`github.com/rs/zerolog`** with **`go.rtnl.ai/x/rlog`** (built on **`log/slog`**). Target **`go.rtnl.ai/x/rlog` v1.12.x** (or current release). This doc is only what you need to do that switch in a new or existing repo—nothing about older rlog APIs.

---

## 1. Find all zerolog usage

- **Imports**: `github.com/rs/zerolog`, `github.com/rs/zerolog/log`.
- **Chains**: `.Info()`, `.Error()`, `.Str(`, `.Err(`, `.Int(`, `.Dur(`, `.Bool(`, `.Time(`, `.Msg(`, `.Msgf(`, `.Fields(`, `.Interface(`, `.Logger` from context.
- **Often missed**: middleware, workers, stores, clients, `init`, tests that match log text or zerolog globals.

Run **`go mod why -m github.com/rs/zerolog`** after edits; if it remains **indirect**, another module still pulls it.

---

## 2. Dependencies

- **Gimlet**: If the app uses **`go.rtnl.ai/gimlet`** logging helpers or middleware, bump **`go.rtnl.ai/x/rlog`** and gimlet in **one** change and fix compile errors together.
- **OpenTelemetry logs**: If you add **`otelslog`**, align **otel**, **`sdk/log`**, and **contrib** versions; one **`go mod tidy`** after **`go.mod`** edits.

---

## 3. Config and levels

Zerolog’s **`zerolog.Level`** and string levels are not the same as **`slog.Level`**.

- Use whatever your config package exposes (often something like **`rlog.LevelDecoder`** or a **`GetLogLevel() → slog.Level`**).
- **Do not** assume level **`.String()`** matches what you used with zerolog (casing/names differ). Tests that compared env vars to **`LogLevel.String()`** should compare to **`slog.LevelInfo`**, **`slog.LevelDebug`**, etc., or to the configured level value.

---

## 4. Zerolog chains → rlog / slog (mechanical)

Zerolog builds a chain, then **`.Msg("…")`**. Rlog uses a **message string** plus **`slog.Attr`** (or key/value pairs on some APIs).

| Zerolog (illustrative) | Use instead |
|------------------------|-------------|
| `log.Info().Str("k", v).Msg("msg")` | `rlog.InfoAttrs(ctx, "msg", slog.String("k", v))` |
| `.Err(err)` | `slog.Any("err", err)` (or `slog.String` if you only log `err.Error()`) |
| `.Int("n", x)` | `slog.Int("n", x)` |
| `.Dur("d", d)` | `slog.Duration("d", d)` |
| `.Bool("b", x)` | `slog.Bool("b", x)` |
| `.Time("t", t)` | `slog.Time("t", t)` |
| `.Interface("k", v)` | `slog.Any("k", v)` |
| `log.With().Str(...)` (derived logger) | `logger.With(slog.String(...))` on **`*rlog.Logger`** — see §6 |

**Levels**: `DebugAttrs`, `InfoAttrs`, `WarnAttrs`, `ErrorAttrs` (and `TraceAttrs` if you use trace). Pass **`ctx`** when you have it so traces and request context propagate.

**Package `log` global** (zerolog): becomes **`rlog.Default()`** or package-level **`rlog.InfoAttrs`**, etc., after you set the default logger (§5).

**Gimlet** `logger.Tracing(c)` (or similar): use the returned logger with **`…Attrs`** methods and **`c.Request.Context()`** (or your span context) as **`ctx`**.

Before mass replace, skim **`go.rtnl.ai/x/rlog`** for exact method names— they differ from zerolog’s fluent names.

---

## 5. Root logger at startup

1. Build an **`slog.Handler`** (e.g. **`slog.NewJSONHandler`**, **`slog.NewTextHandler`**) with the options your app needs (level, format).
2. Wrap with **`rlog.New(slog.New(handler))`** to get **`*rlog.Logger`**.
3. Call **`rlog.SetDefault(thatLogger)`** so **`rlog.Info`**, **`rlog.Default()`**, and the rest of the package API use that logger.

That is the full “global logger” story for this task—no extra steps for “syncing” other defaults.

---

## 6. Loggers with extra fields (replacing zerolog `With`)

Zerolog: **`log.With().Str("component", "x").Logger()`**.

On **`*rlog.Logger`**, use **`l.With(...)`** / **`l.WithGroup(...)`** on the **rlog** value so you still have **`Fatal`**, **`Trace`**, **`…Attrs`**, etc.

Do **not** use only **`l.Logger.With(...)`** on the embedded **`slog.Logger`** for that purpose—it returns **`slog.Logger`**, not **`*rlog.Logger`**, and you lose rlog’s methods.

Shorthand for the default: **`rlog.With(...)`**, **`rlog.WithGroup(...)`** on the package.

---

## 7. Fatal and graceful shutdown

If the code used **`log.Fatal()`** / **`Fatal().Msg(...)`** and you need work before **`os.Exit`** (e.g. flush HTTP server):

- Use **`rlog.SetFatalHook(func() { … })`** **once**, after the values the hook needs exist (e.g. **`*Server`** for **`Shutdown`**).
- In tests, **`rlog.SetFatalHook(nil)`** in **`t.Cleanup`** so **`Fatal*`** does not leave a custom hook for the next test. Parallel tests share one hook—serialize or clean up carefully.

If you do **not** need a hook, **`rlog.Fatal` / `FatalAttrs`** behave like normal fatal logging and exit.

---

## 8. Multiple outputs (e.g. stdout + OpenTelemetry)

Combine handlers so each sink gets correct **`slog.Record`** behavior (clone per sink; **`Enabled`** should allow a record if **any** sink wants it). Options:

- **`rlog.NewFanOut`**, or
- **`slog.NewMultiHandler`** (stdlib; requires a Go version that provides it—check **`go doc log/slog`** in your toolchain).

Wire **OpenTelemetry**’s **`otelslog`** handler only after the log **`LoggerProvider`** exists if logs must export; when telemetry is off, still install a normal root logger so nothing runs with no logger.

---

## 9. Tests

- Prefer **`rlog.NewCapturingTestHandler`** and helpers (**`ParseJSONLine`**, **`ResultMaps`**, **`RecordsAndLines`**, etc.) over copy-pasted slog test harnesses—see rlog docs.
- Assert **level**, **message**, and **attribute keys/values**, not fragile full-line snapshots unless the repo already does.

---

## 10. Context

Use **`ctx`** from **`http.Request`** or jobs for **`…Attrs`** / **`…Context`** calls when available. For APIs with no **`context`**, **`context.Background()`** for logs is a common compromise.

---

## 11. `go mod tidy` and optional code

Tidy keeps any module imported from **any** `.go` file, including behind **`//go:build`** tags (not **`//go:build ignore`**). Remove or isolate files that pull heavy deps you do not want.

---

## 12. Common mistakes

| Mistake | Why |
|--------|-----|
| **`go build` only** | Tests lock in config and logging behavior—run **`go test ./...`**. |
| **`l.Logger.With`** for scoped rlog | You lose **`*rlog.Logger`** methods—use **`l.With`**. |
| **Stable `LogLevel.String()`** across libraries | Compare **`slog.Level`** values in tests, not arbitrary strings. |
| **Silent startup** | Always set a root logger on boot, including “telemetry disabled” paths. |
