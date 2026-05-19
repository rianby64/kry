# Critics and weak points

## What works well

Generics are the right call. Type-safe actions and states catch whole classes of bugs at compile time. The functional-options pattern (`WithFullHistory`, `WithPanicHandler`, etc.) is idiomatic Go and composes cleanly. The `InstanceFSM` interface passed into callbacks is elegant — the callback can trigger further transitions without holding a reference to the outer machine.

**Single `Param` type is a deliberate trade-off, not a flaw.** Go generics cannot express different param types per transition on the same `Apply` method — this is a language constraint, not a library design mistake. Given that constraint, fixing one `Param` type per machine is actually the right call: it keeps the machine's full data contract in one place, and it keeps the history uniform (`[]Param` is easy to iterate, log, serialize, and inspect). The recommended pattern is a union struct (e.g. `CarParam`) with one pointer field per transition that carries data, plus a named constructor per variant (`ForRide(70)`, `ForStop(0.2)`). The constructor functions recover most of the call-site clarity you would get from multiple types, while the union struct makes the machine's input vocabulary explicit and self-documenting.

## What I'd push back on

**`Apply` vs `Event` duality.** This is the biggest friction point. Having two methods to trigger a transition, where one silently becomes unavailable depending on how you defined your transitions, is surprising. `Event()` returning `ErrNotAllowed` because somewhere in your transition table two entries share an action name is a runtime discovery, not a design-time one. A user building a machine incrementally will hit this and not immediately know why. The distinction between "you name the destination" and "the machine knows the destination" is real and useful — but the silent disabling is a footgun. See **TODO** for the proposed resolution.

**The three-callback design.** `Enter`, `EnterNoParams`, and `EnterVariadic` all on the same struct, dispatched at call time based on how many params are passed, is unexpected. The fact that you can define all three on one transition and the machine picks which one runs depending on the *caller* — not the transition definition — breaks the principle of least surprise. A user reading the transition definition can't tell which callback will actually fire without also knowing every call site. It also means a subtle mistake (passing zero params when you meant to pass one) silently calls a different callback. See **TODO** for the proposed resolution.

**`Src`/`SrcFn`/`Dst`/`DstFn` combinations.** The `Transition` struct has four fields for expressing source and destination, and the valid combinations and their routing into different internal maps (`path`, `pathByMatchSrc`, `pathByMatchDst`, `pathMatch`) are only documented in comments. A struct with four partially-interchangeable fields where some combinations are invalid and others route differently is a lot of implicit knowledge to carry. See **TODO** for the proposed resolution.

## Deliberate escape hatches

Some methods bend the rules of the FSM intentionally. They are not design flaws — they are acknowledged escape hatches for situations where a clean solution would be disproportionately complex. The contract is: **use them, but the history will always show it.** A user auditing history can call `HasViolation()` on any `HistoryItem` to scan for entries where an escape hatch was used.

**`IgnoreCurrentTransition()`** — called inside a callback to silently roll back the state change without returning an error. Already recorded in history via `Ignored: bool`. See **TODO** for the missing `ForceState` counterpart.

**`ForceState(state)`** — bypasses transition rules entirely and sets the current state to a registered state; it rejects unregistered states with `ErrUnknown`. Useful when domain logic demands a state jump that the FSM graph does not express. See **TODO** for the history recording that is currently missing for this method.

## Overall

The core model is solid. The multi-FSM loop detection via context is genuinely clever. The rough edges are mostly in the `Transition` struct design and the `Apply`/`Event` split — both fixable without rethinking the whole thing.

## TODO

**Remove `Name` from transition matching; make it a label only.**

The root cause of the `Apply`/`Event` duality is that `Name` (Action) tries to play two roles at once: the classical FSM input/event that drives the machine, and a grouping label for related transitions. When the same `Name` appears with different destinations the machine becomes non-deterministic, `Event()` is silently disabled, and the caller is forced to specify the destination in `Apply` — which means `Name` was never truly driving the selection to begin with.

The resolution is to commit fully to a **destination-driven** model:

- Remove `Name` from the matching logic entirely. The transition is identified solely by the `(current state, destination state)` pair.
- `Apply` becomes `Apply(ctx, Dst, params...)` — the caller says where they want to go, the machine validates it is legal from the current state and executes the callbacks.
- `Event()` disappears — there is no longer a duality, only `Apply`.
- `Name` is demoted to a plain `string` label on `Transition`, used only for history entries and visualization. It is no longer a generic type parameter.
- The FSM signature simplifies from `FSM[Action, State, Param]` to `FSM[State, Param]`.
- Construction must reject two transitions that share the same `(Src, Dst)` pair, since without `Name` there is no way to disambiguate between them. This is not a loss — it is a nudge toward better modelling (different behaviour from the same source to the same destination should be expressed through different destination states or through params).


**Replace the three-callback fields with a single typed `Enter` field.**

The three-callback weakness is not the arity-based dispatch itself — that is a good idea. The problem is that a single `Transition` struct carries up to three callbacks at once and the machine picks one silently at runtime. Reading a transition definition you cannot tell which callback will fire.

The fix: each transition carries exactly one callback, declared explicitly via a constructor function. The arity is encoded in the constructor, not inferred from which optional field is populated. Multiple transitions with the same `(Name, Src, Dst)` are allowed when they serve different arities — the matching key becomes `(current state, Dst, len(params))`.

Replace the three fields with a single `Enter TransitionHandler` field and three constructors:

```
OnEnter(fn func(ctx, instance) error)                     → fires when len(params) == 0
OnEnterWith(fn func(ctx, instance, p Param) error)        → fires when len(params) == 1
OnEnterVariadic(fn func(ctx, instance, p ...Param) error) → fires when len(params) >= 2
```

Usage for the car `Stop` transition across all three arities:

```
{ Name: "Stop", Src: [MovingStraight, MovingLeft, MovingRight], Dst: Stopped,
  Enter: OnEnter(smoothStop) }

{ Name: "Stop", Src: [MovingStraight, MovingLeft, MovingRight], Dst: Stopped,
  Enter: OnEnterWith(stopWithPressure) }

{ Name: "Stop", Src: [MovingStraight, MovingLeft, MovingRight], Dst: Stopped,
  Enter: OnEnterVariadic(stopWithMultipleInstructions) }
```

Three transition definitions, same `(Name, Src, Dst)`, different arity. Each has exactly one callback. Arity matching is strict — no silent fallthrough to variadic. If no transition exists for the given `(current state, Dst, arity)`, the machine returns an error.

**Replace `Src`/`SrcFn`/`Dst`/`DstFn` fields with a builder API using typed constructors.**

The four-field approach on `Transition` requires implicit knowledge of which combinations are valid and which internal map they route to. The fix applies the same constructor principle used for `Enter`: replace the fields with typed constructors and surface the structure through a builder.

The new API introduces a `Build` entry point and two kinds of destination groupings — exact and function-based — each accepting source matchers composed from `kry.From` or `kry.FnFrom`:

```
kry.Build[State, Param](initialState)
    .GoingTo(state,   sources...)   — exact destination, one or more source matchers
    .FnGoingTo(fn,    sources...)   — function-matched destination, one or more source matchers
    .WithFullHistory()
    .WithPanicHandler(handler)
    .Create()

Source matchers:
    kry.From(states...)             — exact source list
    kry.FnFrom(fn)                  — function-matched source

    .On(name, handler)              — attaches a label and a typed callback; chainable for multiple arities
```

The four combinations from the original design expressed in the new API, using the document workflow as example:

```go
import kry "github.com/rianby64/kry"

machine, err := kry.Build[WorkflowState, WorkflowParam](Draft).

    // exact src, exact dst
    GoingTo(InReview1,
        kry.From(Draft).On("Submit", kry.OnEnterWith(notifyReviewer)),
    ).
    GoingTo(Approved,
        kry.From(InReview1, InReview2, InReview3).On("Approve", kry.OnEnterWith(notifyApproval)),
    ).

    // same dst, different arities on the same source
    GoingTo(Rejected,
        kry.From(InReview1, InReview2, InReview3).
            On("Reject", kry.OnEnter(rejectSilently)).
            On("Reject", kry.OnEnterWith(rejectWithReason)),
    ).

    // FnSrc, exact dst
    GoingTo(Draft,
        kry.FnFrom(func(s WorkflowState) bool {
            return s >= InReview1 && s <= InReview3
        }).On("Retract", kry.OnEnter(notifyRetraction)),
    ).

    // exact src, FnDst
    FnGoingTo(func(s WorkflowState) bool {
        return s >= InReview1 && s <= InReview3
    },
        kry.From(Rejected).On("Reopen", kry.OnEnterWith(notifyReopen)),
    ).

    // FnSrc, FnDst
    FnGoingTo(func(dst WorkflowState) bool {
        return dst >= InReview2 && dst <= InReview3
    },
        kry.FnFrom(func(src WorkflowState) bool {
            return src >= InReview1 && src <= InReview2
        }).On("Escalate", kry.OnEnter(logEscalation)),
    ).

    WithFullHistory().
    WithPanicHandler(myPanicHandler).
    Create()
```

Each `GoingTo` / `FnGoingTo` block corresponds directly to one node in the state diagram and its incoming edges. The routing to internal maps becomes a hidden construction detail. Reading any transition, the matching strategy — exact list or function, for both source and destination — is declared explicitly through the constructor used.

**`FnFrom` / `FnGoingTo` are ergonomics, not middleware — resolved by a two-tier rule.**

`FnFrom` and `FnGoingTo` exist for one reason: to avoid writing large explicit state lists. In SIP, for example, a transition that accepts any provisional response would require listing every integer from 101 to 199 with `From`. `FnFrom(func(s) bool { return s >= 101 && s <= 199 })` is just a compact notation for the same thing. The function carries no special semantic weight — it is not a fallback and it is not a catch-all.

The current implementation resolves matches through an implicit priority ordering (exact → SrcFn → DstFn → both), which is the HTTP middleware pattern applied to an FSM. This makes behaviour silently dependent on which other transitions are defined elsewhere in the machine.

**Construction-time overlap detection is not feasible.** `Fn*` accepts arbitrary Go functions. Determining whether two such functions overlap for any input is undecidable in general — equivalent to function satisfiability analysis. Even probing a known state set is incomplete: states that only appear inside `FnFrom`/`FnGoingTo` logic and never as explicit `From` or `GoingTo` arguments are invisible to the builder. The overlap problem cannot be solved ahead of time.

The practical resolution is an explicit two-tier matching rule, documented as part of the API contract:

1. **Exact always beats `Fn*`** — a `GoingTo` + `From` match wins over any `FnGoingTo` or `FnFrom` match, regardless of registration order. Concrete beats general — this is the one priority that feels natural and is never surprising.

2. **Among `Fn*` matches, first registered wins** — when multiple function-based transitions match the same `(current state, destination)` pair, the first one registered fires. The user controls priority by ordering `FnGoingTo` / `FnFrom` blocks from most specific to least specific.

This is still a single-handler guarantee — at most one transition fires per `Apply` call. The ordering resolves ambiguity among the fuzzy cases without cascading through all of them. The SIP example reads naturally under this rule:

```go
GoingTo(Confirmed,                                          // exact — always wins for 200
    kry.From(Response200).On("OK", kry.OnEnter(handleOK)),
).
FnGoingTo(func(s SIPCode) bool { return s >= 101 && s <= 199 }, // first Fn — fires for 101–199
    kry.FnFrom(...).On("Provisional", kry.OnEnter(handleProvisional)),
).
FnGoingTo(func(s SIPCode) bool { return s >= 100 && s <= 999 }, // second Fn — wider fallback
    kry.FnFrom(...).On("AnyResponse", kry.OnEnter(handleAny)),
)
```

Both rules must be stated explicitly in the documentation. The second rule in particular — first registered wins among `Fn*` — must be visible so the user can reason about and control registration order intentionally.

**Add history recording for `ForceState`.**

`IgnoreCurrentTransition()` is already reflected in history via `Ignored: bool`. `ForceState` is already implemented but has no equivalent recording yet. Two changes are needed:

1. Add `Forced bool` and `ForcedTo *State` to `HistoryItem`. `Forced` signals the escape hatch was used. `ForcedTo` records the state the machine was pushed to, separately from `To` (the originally intended destination), so the history tells the full truth:

```
HistoryItem:
  From:     MovingStraight
  To:       Stopped          ← the transition's intended destination
  ForcedTo: &Waiting         ← what ForceState actually set
  Forced:   true
```

2. Add `HasViolation() bool` to `HistoryItem`, returning `Ignored || Forced`. This gives callers a single predicate to scan history for any escape hatch usage without inspecting individual fields.

The escape hatches remain available — they are acknowledged tools for situations where strict FSM rules would be disproportionately costly. The history records them transparently so the author is always aware when and where the rules were bent.
