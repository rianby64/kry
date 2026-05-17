# Critics and weak points

## What works well

Generics are the right call. Type-safe actions and states catch whole classes of bugs at compile time. The functional-options pattern (`WithFullHistory`, `WithPanicHandler`, etc.) is idiomatic Go and composes cleanly. The `InstanceFSM` interface passed into callbacks is elegant — the callback can trigger further transitions without holding a reference to the outer machine.

## What I'd push back on

**`Apply` vs `Event` duality.** This is the biggest friction point. Having two methods to trigger a transition, where one silently becomes unavailable depending on how you defined your transitions, is surprising. `Event()` returning `ErrNotAllowed` because somewhere in your transition table two entries share an action name is a runtime discovery, not a design-time one. A user building a machine incrementally will hit this and not immediately know why. The distinction between "you name the destination" and "the machine knows the destination" is real and useful — but the silent disabling is a footgun.

**The three-callback design.** `Enter`, `EnterNoParams`, and `EnterVariadic` all on the same struct, dispatched at call time based on how many params are passed, is unexpected. The fact that you can define all three on one transition and the machine picks which one runs depending on the *caller* — not the transition definition — breaks the principle of least surprise. A user reading the transition definition can't tell which callback will actually fire without also knowing every call site. It also means a subtle mistake (passing zero params when you meant to pass one) silently calls a different callback.

**The zero-value state restriction.** Not being able to use the zero value as a valid state is a real constraint that shows up as a runtime error, not a compile-time one. The first thing most Go developers reach for with a state enum is `iota` starting at 0. They'll hit `ErrNotAllowed` during construction and need to trace back why. The error message at that point says "destination state is zero value: not allowed" which is fine, but the constraint itself feels arbitrary — it's an implementation detail of the matching logic leaking into the user-facing API.

**`Src`/`SrcFn`/`Dst`/`DstFn` combinations.** The `Transition` struct has four fields for expressing source and destination, and the valid combinations and their routing into different internal maps (`path`, `pathByMatchSrc`, `pathByMatchDst`, `pathMatch`) are only documented in comments. A struct with four partially-interchangeable fields where some combinations are invalid and others route differently is a lot of implicit knowledge to carry.

**`IgnoreCurrentTransition()` as a side effect.** Calling a method on the instance inside a callback to signal "pretend this didn't happen" is a mutable side-channel. Returning a sentinel error (or a typed result) would make the intent visible at the call site and in the function signature.

**Package name.** `kry` is opaque. Nothing about the name suggests finite state machine. Minor, but it matters for discoverability and for reading import paths in unfamiliar code.

## Overall

The core model is solid. The multi-FSM loop detection via context is genuinely clever. The rough edges are mostly in the `Transition` struct design and the `Apply`/`Event` split — both fixable without rethinking the whole thing.
