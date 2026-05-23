<div align="center"><img src="https://github.com/rianby64/kry/blob/main/icon.png?raw=true" /></div>

# kry

A small, generic finite state machine library for Go. Named after my wife Kry.

```sh
go get github.com/rianby64/kry
```

## At a glance

```go
machine, err := kry.Build[State, Param](initialState).
    GoingTo(Approved,
        kry.From[State, Param](InReview).On("Approve", kry.OnEnter(notifyApproval)),
    ).
    GoingTo(Rejected,
        kry.From[State, Param](InReview).On("Reject", kry.OnEnterVariadic(notifyRejection)),
    ).
    WithFullHistory().
    Create()

err = machine.Apply(ctx, Approved, ApprovalParam{Reviewer: "alice"})
```

- **Generic.** `FSM[State comparable, Param any]` — states and params are typed.
- **Single trigger.** `Apply(ctx, dst, params...)` — no second event API to learn.
- **Two arity rules.** `OnEnter` (exactly 1 param, strict) and `OnEnterVariadic` (any number, lenient).
- **Fuzzy matching.** `FnFrom` / `FnGoingTo` accept predicates instead of state lists.
- **History, panic handling, loop detection, escape hatches** all built in.

## Two ways to construct

Both APIs are first-class and produce the same FSM. Pick whichever fits the situation.

### Builder API (recommended)

```go
import kry "github.com/rianby64/kry"

machine, err := kry.Build[WorkflowState, WorkflowParam](Draft).
    // exact src, exact dst
    GoingTo(InReview,
        kry.From[WorkflowState, WorkflowParam](Draft).On("Submit", kry.OnEnter(notifyReviewer)),
    ).

    // multiple sources to one dst
    GoingTo(Approved,
        kry.From[WorkflowState, WorkflowParam](InReview1, InReview2, InReview3).
            On("Approve", kry.OnEnter(notifyApproval)),
    ).

    // FnSrc, exact dst — "any in-review state can be retracted"
    GoingTo(Draft,
        kry.FnFrom[WorkflowState, WorkflowParam](func(s WorkflowState) bool {
            return s >= InReview1 && s <= InReview3
        }).On("Retract", kry.OnEnterVariadic(notifyRetraction)),
    ).

    // exact src, FnDst — "Rejected can reopen to any review level"
    FnGoingTo(func(s WorkflowState) bool {
        return s >= InReview1 && s <= InReview3
    },
        kry.From[WorkflowState, WorkflowParam](Rejected).On("Reopen", kry.OnEnter(notifyReopen)),
    ).

    // no-op transition (no callback)
    GoingTo(Archived,
        kry.From[WorkflowState, WorkflowParam](Approved, Rejected).
            On("Archive", kry.NoOp[WorkflowState, WorkflowParam]()),
    ).

    WithFullHistory().
    WithPanicHandler(myPanicHandler).
    Create()
```

Each `GoingTo` / `FnGoingTo` block corresponds to one node in the state diagram and its incoming edges.

### Raw transition list

The original `New()` constructor is still available for cases where you want to build the transition table programmatically:

```go
machine, err := kry.New(Draft, []kry.Transition[WorkflowState, WorkflowParam]{
    {
        Name:  "Submit",
        Src:   []WorkflowState{Draft},
        Dst:   InReview,
        Enter: kry.OnEnter(notifyReviewer),
    },
    // ...
}, kry.WithFullHistory[WorkflowParam]())
```

## Callbacks: `OnEnter` vs `OnEnterVariadic`

Each transition carries exactly one callback. The constructor name declares the arity.

| Constructor | Fires when | Wrong-arity call |
|---|---|---|
| `kry.OnEnter(fn)` | `len(params) == 1` | returns `ErrNotAllowed` |
| `kry.OnEnterVariadic(fn)` | any number of params | passes them all through |
| `kry.NoOp[State, Param]()` | always, does nothing | accepts any params |

**Arity is enforced strictly for `OnEnter`.** Calling `Apply(ctx, dst)` with zero or two params against an `OnEnter` transition returns `ErrNotAllowed` and the state does not change. This catches misuse at the call site instead of silently dropping params.

```go
// strict — must be called with exactly one ApprovalParam
kry.OnEnter(func(ctx context.Context, fsm kry.InstanceFSM[State, ApprovalParam], p ApprovalParam) error {
    return notify(p.Reviewer)
})

// lenient — works with zero, one, or many params
kry.OnEnterVariadic(func(ctx context.Context, fsm kry.InstanceFSM[State, ApprovalParam], p ...ApprovalParam) error {
    for _, x := range p { _ = x }
    return nil
})
```

## ⚠ Matching priority: how `From`, `FnFrom`, `GoingTo`, `FnGoingTo` resolve

When `Apply(ctx, dst, ...)` runs, the dispatcher resolves the matching transition in **strict order**:

1. **Exact `From` + exact `GoingTo`** — always wins.
2. **`FnFrom` + exact `GoingTo`** — first matching wins, in registration order.
3. **Exact `From` + `FnGoingTo`** — first matching wins, in registration order.
4. **`FnFrom` + `FnGoingTo`** — first matching wins, in registration order.

Two rules to internalize:

> **Rule 1 — exact always beats `Fn*`.** A concrete `(From, GoingTo)` pair wins over any predicate-based match, regardless of registration order. Concrete beats general.

> **Rule 2 — among `Fn*` matches, registration order is the priority.** The first registered `Fn*` block whose predicates return `true` fires. Order them **most specific first**, fall-throughs last.

Construction-time overlap detection between predicates is not feasible (function equivalence is undecidable in general), so the registration order is the contract.

```go
// SIP-style example — exact wins, then narrow fn, then wide fn
kry.Build[SIPCode, SIPParam](Idle).
    // exact — always wins for 200
    GoingTo(Confirmed,
        kry.From[SIPCode, SIPParam](Code200).On("OK", kry.OnEnterVariadic(handleOK)),
    ).
    // first Fn — fires for 101–199
    FnGoingTo(
        func(s SIPCode) bool { return s >= 101 && s <= 199 },
        kry.FnFrom[SIPCode, SIPParam](anyState).On("Provisional", kry.OnEnterVariadic(handleProvisional)),
    ).
    // second Fn — wider fallback, only reached if the first didn't match
    FnGoingTo(
        func(s SIPCode) bool { return s >= 100 && s <= 999 },
        kry.FnFrom[SIPCode, SIPParam](anyState).On("AnyResponse", kry.OnEnterVariadic(handleAny)),
    )
```

### Uniqueness constraint

- **Exact `(From, GoingTo)` paths** allow **at most one handler per pair**, regardless of arity. A duplicate registration returns `ErrRepeated` at construction time.
- **`Fn*` paths** have **no uniqueness constraint** — register as many as you like and order them deliberately.

## Single `Param` type per machine

Go generics cannot express different param types per transition on the same `Apply` method. `kry` fixes one `Param` type per machine and recommends a union struct with named constructors:

```go
type CarParam struct {
    Speed    *float64
    Pressure *float64
}

func ForRide(speed float64) CarParam   { return CarParam{Speed: &speed} }
func ForStop(p float64) CarParam       { return CarParam{Pressure: &p} }

err := car.Apply(ctx, Moving, ForRide(70))
err  = car.Apply(ctx, Stopped, ForStop(0.2))
```

Call sites stay readable, history items remain uniform (`[]CarParam` is easy to log/serialize), and each transition's expected payload is documented by which constructor builds it.

## History

```go
machine, _ := kry.Build[State, Param](initial).
    GoingTo(...).
    WithFullHistory().                  // unbounded
    // WithHistory(50).                  // bounded
    // WithEnabledStackTrace().          // capture stack traces on errors
    // WithCloneHandler(deepCopyParams). // custom param cloning (e.g. CBOR)
    Create()

for _, item := range machine.History() {
    fmt.Printf("%s %v→%v err=%v ignored=%v forced=%v\n",
        item.Name, item.From, item.To, item.Err, item.Ignored, item.Forced)
}
```

Each `HistoryItem` carries: `Name`, `From`, `To`, `Params`, `Err`, `StackTrace`, `Reason`, `Ignored`, `Forced`, `ForcedTo`, `ExpectFailed`.

Nested `Apply` calls inside a callback are recorded chronologically under the parent — the parent item appears first, then its children.

## Escape hatches (and why history records them)

Some methods deliberately bend FSM rules. Both are tracked in history so the author always sees when and where the rules were bent.

```go
// Inside an OnEnter / OnEnterVariadic callback:
fsm.IgnoreCurrentTransition()    // silently roll back this transition (no error)
err := fsm.ForceState(SomeState) // jump to any registered state, bypassing transitions
```

History flags:

| Flag | Meaning |
|---|---|
| `item.Ignored` | `IgnoreCurrentTransition()` was called during this transition |
| `item.Forced` | `ForceState(...)` was called during this transition |
| `item.ForcedTo` | the state `ForceState` actually set (distinct from `item.To`, the originally intended destination) |
| `item.HasViolation()` | `Ignored \|\| Forced` — single predicate for auditing |

```go
for _, item := range machine.History() {
    if item.HasViolation() {
        log.Printf("escape hatch used: %+v", item)
    }
}
```

`ForceState` rejects unregistered states with `ErrUnknown` and can also be called **outside** an `Apply` — in which case a standalone history item is recorded.

### ⚠ Known sharp edge: `IgnoreCurrentTransition` and nested `Apply`

`IgnoreCurrentTransition()` and a nested `Apply()` inside the same callback interact in ways that are likely to surprise:

- **Calling `IgnoreCurrentTransition()` *after* the nested `Apply`** is silently dropped — the inner Apply's defer clears the "running" flag that `IgnoreCurrentTransition` guards on. The outer transition is **not** rolled back.
- **Calling `IgnoreCurrentTransition()` *before* the nested `Apply`** is consumed by the inner call — the inner transition rolls back and is marked `Ignored`, but the outer proceeds normally.

Until this is fixed, **don't mix `IgnoreCurrentTransition()` with nested `Apply()` calls in the same callback.** See `critics-weak-points.md` for the planned resolution.

## Panic recovery

Provide a `WithPanicHandler` to intercept callback panics. State rolls back, history records the panic with stack trace, and `Apply` returns `nil`:

```go
machine, _ := kry.Build[State, Param](initial).
    GoingTo(...).
    WithPanicHandler(func(ctx context.Context, panicReason any) {
        log.Printf("callback panicked: %v", panicReason)
    }).
    WithFullHistory().
    WithEnabledStackTrace().
    Create()
```

Without a panic handler the panic is re-thrown after the state is rolled back — the FSM does not swallow panics silently.

## Loop detection

`kry` carries a per-machine loop tracker on the `context.Context`. If a callback chain attempts the same `(machine, from, to)` transition twice in the same context tree, `Apply` returns `ErrLoopFound`.

The tracker is keyed by machine ID, so **multiple FSMs can call into each other** through the same context without false positives — only a true cycle in one machine trips it.

```go
machine1.Apply(ctx, open)  // calls machine2.Apply(ctx, open)
                           // which calls machine1.Apply(ctx, open) → ErrLoopFound
```

## Decorators: `Expect*`

Assert that a specific handler fired during `Apply`. Useful in tests where you want to verify *which* transition resolved without inspecting state directly:

```go
err := machine.
    With(kry.ExpectEnterVariadic(handlerClose)).
    Apply(ctx, open)

// The history item carries ExpectFailed=true if a different handler fired.
```

## Visualization

Two helpers produce Graphviz DOT output:

```go
kry.VisualizeStateLinks(transitions)  // edges
kry.VisualizeActions(transitions)     // subgraphs grouped by transition name
```

`fsm.String()` returns the full digraph for the constructed machine.

## Errors

| Error | When |
|---|---|
| `kry.ErrNotFound` | no transition matches the requested `(current, dst)` |
| `kry.ErrUnknown` | `ForceState` was called with an unregistered state |
| `kry.ErrRepeated` | two transitions registered for the same exact `(Src, Dst)` |
| `kry.ErrLoopFound` | the same transition was attempted twice in the same context |
| `kry.ErrNotAllowed` | `OnEnter` (strict) called with a number of params ≠ 1 |

## Concurrency

`kry` does **not** lock the machine for you. If a single FSM is shared across goroutines, synchronize at the call site (e.g. wrap `Apply` with a mutex). The internal history keeper *is* concurrency-safe.

## Requirements

- Go 1.24 or newer.

## Examples

See [`example/open-close`](./example/open-close) for the smallest possible machine, and [`example/elevator`](./example/elevator) for a typed-state, typed-param walkthrough.

## License

MIT.
