# golog

This logging package provides a standardized way to initialize [zerolog](https://github.com/rs/zerolog) based loggers. It also integrates with the [ctxerr](https://github.com/KernelPryanic/ctxerr) package to propagate additional error contextual information up the call stack.

## Usage

```go
log.SetTimeFormat("2006-01-02 15:04:05")
log.SetLogLevel("info")
logger := log.New(true) // true for console writer
logger.Info().Str("some_key", "some_value").Msg("hello world")
```

```go
func funcA(ctx context.Context, ...) {
    logger := log.Default.With().Str("some_base_id", id).Logger() // this info will be always displayed in the logs of child functions (propagated down the call stack)
    ctx = logger.WithContext(ctx) // put the logger into the context
    if err := funcB(ctx); err != nil {
        logger.Error().Ctx(ctxerr.Ctx(ctx, err)).Str("document", largeDocument).Msg("an error occurred") // stuff the context with the context error we got from down the call stack
    }
    ...
}

func funcB(ctx context.Context) error {
    ...
    if err := funcC(ctx); err != nil {
        return ctxerr.With(err, map[string]any{"manifest": largeManifest}) // additional contextual information to include only when there's an error and propagate it up the call stack
    }
    return nil
}
```

**Note:**

- It's important to use the `ctxerr.Ctx(ctx, err)` function to add context to the logger and `ctxerr.With(err, map[string]any{...})` to propagate additional contextual information up the call stack.
- `SetTimeFormat` must be called before `New` to take effect.
