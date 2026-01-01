# Requirements

1. **Separation of concerns**
   Keep argument/flag parsing at the edges. Commands should delegate to exported functions that contain the business logic (MVC-inspired: parsing ≠ logic).

2. **Command tree wiring**
   The root command registers its direct children (e.g., `rootCmd.AddCommand(log.LogCmd)`), and each child registers its own subcommands.

3. **Pragmatic structure**
   Avoid over-engineering: no DI frameworks and don’t create `internal/*` packages for every command. Place shared logic in importable modules so it’s reusable and testable without duplication.

4. **Dependency policy**
   Prefer native/standard library solutions over third-party packages whenever possible.

5. **Testing**
   For tests, use the `asserts` library instead of `if/fail` conditions.

6. **Backwards compatibility**
   This is not a public library; legacy compatibility isn’t required.

7. **CLI UX**
   Add colorized output to make the CLI more user-friendly.

8. **Comments**
   Do not not add comments unless specifically instructed
