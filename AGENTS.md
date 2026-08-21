# Project conventions

- Keep concrete stateful implementations unexported and expose them through small interfaces.
- Constructors for those implementations return the interface, usually together with an error.
- Prefer passing and returning domain structs by value in public APIs. Do not expose implementation pointers unless an external API makes that unavoidable.
- Keep UI concerns separate from domain models and persistence.
- Use structured concurrency exclusively. Do not start unowned or fire-and-forget goroutines.
- Put concurrent work behind a lifecycle method such as `Run(ctx, ...)`. It may start child goroutines through a structured mechanism such as `errgroup`, but it must always cancel and wait for every goroutine it starts before returning.
- The logbook must support multiple operating modes. Do not encode the currently common modes as a closed database constraint or Go enum.
- Keep the README focused on what the program is, setup, and user and developer workflows needed to build, run, test, or modify it.
- Record technical decisions and implementation details in `ROADMAP.md`, not in the README.
- Track completed and future technical work in `ROADMAP.md` with Markdown checkboxes: `[x]` for completed work and `[ ]` for pending work.
- Update `ROADMAP.md` when a technical decision is made, implemented, superseded, or added as future work.
- Implement the change agreed with the user without broadening its scope, introducing unrequested abstractions, or refactoring adjacent code. If the requested change or its boundary is unclear, ask before choosing an interpretation.
- Implement the change that was agreed with the user without broadening its scope, introducing unrequested abstractions, or refactoring adjacent code. If the requested change or its boundary is unclear, ask the user before choosing an interpretation.
