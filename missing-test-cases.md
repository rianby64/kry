# Missing test cases

## `ForceState`

- **Unknown state** — `ForceState()` with a state not registered in any transition should return `ErrUnknown`. The code exists (`fsm.go:162-172`) but there is no test for it.
- **Called outside a callback** — `Test_force_state` only tests `ForceState()` from inside an `Enter` callback. Calling it directly on the machine (outside any apply) is not tested.

## `Previous()`

- **Fresh FSM** — `Previous()` on a newly created machine before any transition should return the zero value of `State`. Not tested.
- **After a failed `Apply`** — `Previous()` should remain unchanged when `Apply` returns an error (state rolls back). Not tested explicitly (the failing-callback tests only check `Current()`).

## `Event()`

- **With params** — every test that calls `Event()` passes no params. Passing one or more params through `Event()` is not tested.
- **Current state not in any valid `Src` for the action** — `Event()` resolves to a fixed `Dst` and then delegates to `Apply`. When the current state is not a valid source for that transition, `Apply` returns `ErrNotFound`. This path through `Event()` is untested (only `ErrUnknown` and `ErrNotAllowed` paths are covered).

## `IgnoreCurrentTransition()`

- **Inside `Enter` or `EnterVariadic`** — `Test_ignore_transition_ok` only exercises it from `EnterNoParams`. The same behaviour inside the single-param and variadic callback forms is not tested.
- **Combined with nested `Apply`** — calling `IgnoreCurrentTransition()` in a callback that also calls `Apply()` internally is not tested (what happens to both the outer and inner history entries?).

## Loop detection

- **Cross-machine loop (M1→M2→M1)** — `Test_loop_case_infinity_break_two_machines` tests M1→M2 without a loop back. M1 triggering M2 which then triggers M1 again is the missing case; the context-based per-ID tracking should catch this, but there is no test for it.

## `WithCloneHandler`

- **Happy path** — a custom clone handler is never exercised in any test. There is no verification that params stored in history are produced by the custom handler.
- **Error path** — a custom clone handler returning an error should propagate back through `intermediateKeeper` → `apply` → `Apply`. This code path has no test.

## Panic handling

- **No handler set** — `Test_panic_case1` always uses `WithPanicHandler`. When no handler is provided and a callback panics, the recovered error should be re-panicked. The re-panic path (`fsm.go:266-270`) has no test.
- **Panic history entry error field** — `Test_panic_case1` checks `StackTrace` and basic fields but does not assert that `history[1].Err` contains the panic reason as an error.

## `pathMatch` (both `SrcFn` + `DstFn`)

- **Callback error** — `Test_transit_match_dst_case2` only tests the success path. A callback returning an error on a `SrcFn+DstFn` matched transition is not tested, and the expected state rollback is not verified.
- **No matching state** — calling `Apply` with an action that has only `SrcFn+DstFn` transitions but neither function matches is not tested.

## History edge cases

- **Disabled history (default, `size=0`)** — `History()` should return an empty slice. Not tested explicitly; all history tests opt in with `WithFullHistory` or `WithHistory(n)`.
- **Size-limited history with nested `Apply` chains** — `WithHistory(n)` combined with a chain of nested applies is not tested; only the unit-level keeper test and full-history machine tests cover nested chains.
- **`ExpectFailed` + `Ignored` in the same item** — calling `IgnoreCurrentTransition()` inside a callback while an `Expect*` decorator is also active is not tested.

## `Event()` when disabled

- `Test_set_transitions_retrigger_ok` verifies that `Event()` returns `ErrNotAllowed` for one specific action when `canTriggerEvents` is false. There is no test verifying that every action is blocked in that mode (e.g., calling `Event` with an action that would otherwise have been valid in a single-Dst machine).
