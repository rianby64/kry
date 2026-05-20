# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run Test_set_event_ok ./...

# Run tests with verbose output
go test -v ./...

# Build the module and examples
go build ./...
```

## Status

This repository is mid-transition from v1 to v2. The design decisions for v2 are recorded in `critics-weak-points.md`. The remainder of this file describes the **v2 target architecture**.

**Completed so far:**
- `Action` type parameter removed. Current signature: `FSM[State comparable, Param any]`.
- `Event()` removed. `Apply(ctx, dst, params...)` is the only trigger.
- `Name` demoted to a plain `string` label stored in `callbacks` and `HistoryItem`.
- `HistoryItem.Name string` replaces the former `Action` field.

**Still pending:** builder API (`Build/GoingTo/FnGoingTo/From/FnFrom/.On`), single-field `Enter TransitionHandler` with arity constructors, `ForceState` history recording (`Forced`/`ForcedTo`/`HasViolation`).

## v2 Architecture

`kry` is a generic finite state machine (FSM) library. Current signature:

```
FSM[State comparable, Param any]
```

### Construction

The `New()` + `[]Transition{}` API is replaced by a builder:

```go
machine, err := kry.Build[State, Param](initialState).
    GoingTo(dst, sources...).
    FnGoingTo(dstFn, sources...).
    WithFullHistory().
    WithPanicHandler(handler).
    Create()
```

Source matchers passed to `GoingTo` / `FnGoingTo`:

```go
kry.From(states...)      // exact source list
kry.FnFrom(fn)           // function-matched source
```

Each source matcher is chained with `.On(name, handler)` to attach a label and a typed callback. Multiple `.On()` calls on the same source express different arities for the same transition:

```go
kry.From(Draft).
    On("Submit", kry.OnEnter(notifyReviewer)).          // len(params) == 0
    On("Submit", kry.OnEnterWith(notifyWithReason)).    // len(params) == 1
    On("Submit", kry.OnEnterVariadic(notifyAll))        // len(params) >= 2
```

### Transition model

- A transition is identified by `(current state, destination state, arity)`, not by action name.
- `Apply(ctx, dst, params...)` is the only trigger method — `Event()` is removed.
- Construction rejects two transitions with the same `(src, dst)` pair and the same arity.

### Callback constructors

Replace the three struct fields with typed constructors that encode arity explicitly:

| Constructor | Fires when |
|---|---|
| `kry.OnEnter(fn)` | `len(params) == 0` |
| `kry.OnEnterWith(fn)` | `len(params) == 1` |
| `kry.OnEnterVariadic(fn)` | `len(params) >= 2` |

No silent fallthrough between arities — missing arity returns an error.

### Fn* matching rule

`FnFrom` and `FnGoingTo` are compact notation for large state ranges, not middleware or fallbacks.

Resolution order per `Apply` call:
1. **Exact beats Fn*** — a `GoingTo` + `From` match wins over any `FnGoingTo` or `FnFrom` match.
2. **Among Fn* matches, first registered wins** — order `FnGoingTo` / `FnFrom` blocks from most specific to least specific.

Construction-time overlap detection for Fn* is not feasible (determining whether two arbitrary Go functions overlap is undecidable). The two-tier rule is the documented contract.

### Escape hatches

Both methods are intentional but must be visible in history:

- **`IgnoreCurrentTransition()`** — rolls back the state change inside a callback without returning an error. Recorded as `Ignored: bool` in `HistoryItem`.
- **`ForceState(state)`** — sets the current state to any registered state, bypassing transition rules. Rejects unregistered states with `ErrUnknown`. Needs `Forced: bool` and `ForcedTo: *State` added to `HistoryItem`.

`HistoryItem.HasViolation()` returns `Ignored || Forced` — a single predicate for auditing escape hatch usage.

### Single Param type

Each machine fixes one `Param` type. For transitions that carry different data, the recommended pattern is a union struct with one pointer field per variant and named constructors:

```go
type CarParam struct {
    Speed    *float64
    Pressure *float64
}
func ForRide(speed float64) CarParam    { return CarParam{Speed: &speed} }
func ForStop(p float64) CarParam        { return CarParam{Pressure: &p} }
```

This is a language constraint, not a design flaw. The union struct keeps the machine's input vocabulary explicit and makes history uniform.

### Core files

- **`fsm.go`** — `FSM[State, Param]`, `InstanceFSM` interface, `Transition` struct, `New()` constructor.
- **`construct_transitions.go`** — Parses transitions into four internal maps (`path`, `pathByMatchSrc`, `pathByMatchDst`, `pathMatch`). Called once at construction; immutable after.
- **`apply.go`** — `Apply(ctx, dst, params...)`. Resolution priority: exact → SrcFn → DstFn → both-Fn. No `Event()`.
- **`check_loop.go`** — Loop detection in `context.Context`, keyed by FSM `uint64` ID.
- **`options.go`** — Functional options and `Expect*` decorator functions.
- **`history.go`** — `HistoryItem[State, Param]` with `Name string` field, and `historyKeeper` (singly-linked list, optional size cap).
- **`expect_handlers.go`** — Compares expected vs. actual callback function pointers.
- **`handlers.go`** — `OnEnter`, `OnEnterWith`, `OnEnterVariadic` constructors (v2 builder target).
- **`viz.go`** — Graphviz DOT output via `VisualizeActions` / `VisualizeStateLinks`.
