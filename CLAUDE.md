# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run Test_set_transitions_string_int_ok ./...

# Run tests with verbose output
go test -v ./...

# Build the module and examples
go build ./...
```

## Architecture

`kry` is a generic finite state machine (FSM) library. The three type parameters used throughout are `Action` (comparable), `State` (comparable), and `Param` (any).

### Core files

- **`fsm.go`** — Defines `FSM[A,S,P]`, the `InstanceFSM` interface, and the `Transition` struct. `New()` is the only constructor; it takes an initial state, a slice of transitions, and option funcs.
- **`construct_transitions.go`** — Parses the transition slice into four internal lookup maps keyed by action and state. Called once at construction; the structure is immutable after that.
- **`apply.go`** — `Apply()` and `Event()` are the two public methods to trigger transitions. `Apply()` requires an explicit target state; `Event()` derives the target from the transition definition (disabled if any action maps to multiple destinations). Resolution priority inside `Apply()`: exact match → SrcFn match → DstFn match → both-Fn match.
- **`check_loop.go`** — Loop detection stored in `context.Context`. Each FSM tracks its own transitions by a unique `uint64` ID embedded as the context key, so two FSMs sharing a context do not interfere with each other's loop counters.
- **`options.go`** — Functional options (`WithHistory`, `WithFullHistory`, `WithStackTrace`, `WithPanicHandler`, `WithCloneHandler`) and the `Expect*` decorator functions. `With(opts...)` returns `InstanceFSM`, enabling one-shot decorator chaining without mutating the FSM permanently.
- **`history.go`** — `HistoryItem` and an internal singly-linked list (`historyKeeper`) with optional size cap. History is disabled by default (size 0); enabled with `WithFullHistory` or `WithHistory(n)`.
- **`expect_handlers.go`** — `checkCallbacksAgainstExpectHandlers()` compares function pointers of expected vs. actual callbacks, setting `ExpectFailed` in the history item when they don't match.
- **`viz.go`** — `VisualizeActions` / `VisualizeStateLinks` produce Graphviz DOT output. `FSM.String()` returns a `digraph` block generated at construction time.

### Callback dispatch

Each `Transition` can carry up to three callback fields: `EnterNoParams`, `Enter`, and `EnterVariadic`. The FSM dispatches based on the number of params passed at call time: zero → `EnterNoParams`, one → `Enter`, two or more → `EnterVariadic`. If the exact-arity callback is nil, it falls back to `EnterVariadic`.

### Key invariants

- State zero value must not be used as a valid state; `Dst: 0` (or equivalent zero) is rejected at construction with `ErrNotAllowed`.
- `Event()` is disabled (`ErrNotAllowed`) when any action name appears in multiple transitions with different `Dst` values — use `Apply()` instead.
- `IgnoreCurrentTransition()` silently rolls the state back inside a callback without returning an error; it is a no-op when called outside an active `apply`.
- Nested `Apply()` calls inside callbacks (chaining transitions) are valid; loop detection prevents re-entering the same `from→to` pair in the same FSM.
