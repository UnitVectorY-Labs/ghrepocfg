# Development requirements

- Write idiomatic Go; prefer the standard library.
- Keep `docs/` current with a concise, marketing-oriented `README.md` and clear reference documentation for supported behavior.
- Export and apply must degrade gracefully on permission denial: continue independent work and log accurate, concise warnings identifying skipped attributes. Never treat unreadable state as empty, false, or disabled.
- Keep `--strict` optional and off by default. Strict export requires complete requested state; strict apply succeeds only when all managed values are applied and verified. Do not imply atomicity or rollback.
- Test meaningful behavior changes and run the relevant checks.
