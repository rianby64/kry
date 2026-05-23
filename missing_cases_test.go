package kry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── ForceState ───────────────────────────────────────────────────────────────

func Test_force_state_unknown_outside_apply(t *testing.T) {
	const (
		closed int = iota + 1
		open
		unregistered
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
	})

	require.NoError(t, err)
	require.ErrorIs(t, machine.ForceState(unregistered), ErrUnknown)
	require.Equal(t, closed, machine.Current())
}

func Test_force_state_unknown_inside_apply(t *testing.T) {
	const (
		closed int = iota + 1
		open
		unregistered
	)

	machine, err := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(_ context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.ForceState(unregistered)
			}),
		},
	})

	require.NoError(t, err)
	err = machine.Apply(t.Context(), open)
	require.ErrorIs(t, err, ErrUnknown)
	// State rolls back on callback error
	require.Equal(t, closed, machine.Current())
}

// ── IgnoreCurrentTransition ──────────────────────────────────────────────────

func Test_ignore_transition_inside_OnEnter(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, err := New(closed, []Transition[int, int]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnter(func(_ context.Context, instance InstanceFSM[int, int], _ int) error {
				instance.IgnoreCurrentTransition()
				return nil
			}),
		},
	}, WithFullHistory[int]())

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open, 1))
	require.Equal(t, closed, machine.Current())

	history := machine.History()
	require.Len(t, history, 1)
	require.True(t, history[0].Ignored)
	require.True(t, history[0].HasViolation())
}

func Test_ignore_transition_called_after_nested_apply_is_noop(t *testing.T) {
	// Documents current behaviour: when a callback calls a nested Apply *and then*
	// IgnoreCurrentTransition(), the ignore is silently skipped. The inner Apply's
	// defer resets fsk.runningApply to false, so the subsequent
	// IgnoreCurrentTransition() (which guards on runningApply) becomes a no-op.
	const (
		closed int = iota + 1
		open
		roger
	)

	machine, err := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				if errInner := instance.Apply(ctx, roger); errInner != nil {
					return errInner
				}
				instance.IgnoreCurrentTransition() // silently dropped
				return nil
			}),
		},
		{Name: "roger", Src: []int{open}, Dst: roger},
	}, WithFullHistory[any]())

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))

	// State is whatever the inner apply left it as — outer ignore had no effect.
	require.Equal(t, roger, machine.Current())

	history := machine.History()
	require.Len(t, history, 2)
	for _, item := range history {
		require.False(t, item.Ignored)
		require.False(t, item.HasViolation())
	}
}

func Test_ignore_transition_called_before_nested_apply_propagates_to_inner(t *testing.T) {
	// Documents current behaviour: when a callback calls IgnoreCurrentTransition()
	// *before* a nested Apply, the inner Apply's defer consumes the ignore flag —
	// the inner transition rolls back, and the outer no longer sees the flag set.
	const (
		closed int = iota + 1
		open
		roger
	)

	machine, err := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				instance.IgnoreCurrentTransition() // flag set
				return instance.Apply(ctx, roger)  // inner defer consumes the flag
			}),
		},
		{Name: "roger", Src: []int{open}, Dst: roger},
	}, WithFullHistory[any]())

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))

	// Inner rolled back due to the ignore flag; state ends at the outer destination.
	require.Equal(t, open, machine.Current())

	history := machine.History()
	require.Len(t, history, 2)

	// Outer item is recorded first (parent-before-children ordering),
	// with Ignored=false because the inner defer consumed the flag.
	require.Equal(t, "open", history[0].Name)
	require.False(t, history[0].Ignored)

	// Inner item carries Ignored=true (consumed the flag).
	require.Equal(t, "roger", history[1].Name)
	require.True(t, history[1].Ignored)
	require.True(t, history[1].HasViolation())
}

// ── Loop detection across machines ───────────────────────────────────────────

func Test_loop_case_cross_machine_loop(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	var (
		machine1 *FSM[int, any]
		machine2 *FSM[int, any]
	)

	machine1, _ = New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, _ InstanceFSM[int, any], _ ...any) error {
				return machine2.Apply(ctx, open)
			}),
		},
		{Name: "close", Src: []int{open}, Dst: closed},
	})

	machine2, _ = New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, _ InstanceFSM[int, any], _ ...any) error {
				// Move M1 back to closed, then try the same (closed→open) again.
				// The per-machine loop detector should catch this on M1.
				if errClose := machine1.Apply(ctx, closed); errClose != nil {
					return errClose
				}
				return machine1.Apply(ctx, open)
			}),
		},
	})

	require.ErrorIs(t, machine1.Apply(t.Context(), open), ErrLoopFound)
}

// ── WithCloneHandler ─────────────────────────────────────────────────────────

func Test_clone_handler_happy_path(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	cloneCalls := 0
	cloner := func(params ...string) ([]string, error) {
		cloneCalls++
		cloned := make([]string, len(params))
		for i, p := range params {
			cloned[i] = "cloned:" + p
		}
		return cloned, nil
	}

	machine, err := New(closed, []Transition[int, string]{
		{Name: "open", Src: []int{closed}, Dst: open},
	}, WithFullHistory[string](), WithCloneHandler(cloner))

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open, "original"))

	history := machine.History()
	require.Len(t, history, 1)
	require.Equal(t, []string{"cloned:original"}, history[0].Params)
	require.Positive(t, cloneCalls)
}

func Test_clone_handler_error_path(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	cloneErr := errors.New("clone failed")
	cloner := func(_ ...string) ([]string, error) {
		return nil, cloneErr
	}

	machine, err := New(closed, []Transition[int, string]{
		{Name: "open", Src: []int{closed}, Dst: open},
	}, WithFullHistory[string](), WithCloneHandler(cloner))

	require.NoError(t, err)

	err = machine.Apply(t.Context(), open, "p")
	require.Error(t, err)
	require.ErrorIs(t, err, cloneErr)
}

// ── Panic handling ───────────────────────────────────────────────────────────

func Test_panic_no_handler_repanics(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, _ := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				panic("boom")
			}),
		},
	}, WithFullHistory[any]())

	defer func() {
		r := recover()
		require.NotNil(t, r, "expected re-panic when no panic handler is set")
		// State should have rolled back
		require.Equal(t, closed, machine.Current())
	}()

	_ = machine.Apply(t.Context(), open)
	t.Fatal("Apply did not panic without a panic handler")
}

func Test_panic_history_err_contains_reason(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, _ := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				panic("boom!")
			}),
		},
	},
		WithPanicHandler[any](func(_ context.Context, _ any) {}),
		WithFullHistory[any](),
	)

	require.NoError(t, machine.Apply(t.Context(), open))

	history := machine.History()
	require.Len(t, history, 1)
	require.Error(t, history[0].Err)
	require.Equal(t, "boom!", history[0].Err.Error())
}

// ── pathMatch (SrcFn + DstFn) ────────────────────────────────────────────────

func Test_pathmatch_callback_error_rolls_back(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	calledErr := errors.New("path-match boom")

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
		{
			Name: "path",
			SrcFn: func(s int) bool { return s == open },
			DstFn: func(s int) bool { return s == closed },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				return calledErr
			}),
		},
	})

	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	err = machine.Apply(t.Context(), closed)
	require.ErrorIs(t, err, calledErr)
	// State rolled back to open
	require.Equal(t, open, machine.Current())
}

func Test_pathmatch_no_matching_state_returns_not_found(t *testing.T) {
	const (
		closed int = iota + 1
		open
		other
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
		{
			Name: "path",
			// SrcFn only matches `open`, DstFn only matches `other`
			SrcFn: func(s int) bool { return s == open },
			DstFn: func(s int) bool { return s == other },
		},
	})

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))

	// Apply with a dst that no SrcFn+DstFn pair matches → ErrNotFound
	err = machine.Apply(t.Context(), closed)
	require.ErrorIs(t, err, ErrNotFound)
	require.Equal(t, open, machine.Current())
}

// ── History edge cases ───────────────────────────────────────────────────────

func Test_history_disabled_returns_empty(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
	})

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))
	require.Empty(t, machine.History())
}

func Test_history_size_limited_with_nested_apply(t *testing.T) {
	const (
		closed int = iota + 1
		mid1
		mid2
		mid3
		mid4
		mid5
	)

	machine, err := New(closed, []Transition[int, any]{
		{
			Name: "to-mid1", Src: []int{closed}, Dst: mid1,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.Apply(ctx, mid2)
			}),
		},
		{
			Name: "to-mid2", Src: []int{mid1}, Dst: mid2,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.Apply(ctx, mid3)
			}),
		},
		{
			Name: "to-mid3", Src: []int{mid2}, Dst: mid3,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.Apply(ctx, mid4)
			}),
		},
		{
			Name: "to-mid4", Src: []int{mid3}, Dst: mid4,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.Apply(ctx, mid5)
			}),
		},
		{Name: "to-mid5", Src: []int{mid4}, Dst: mid5},
	}, WithHistory[any](2))

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), mid1))
	require.Equal(t, mid5, machine.Current())

	history := machine.History()
	require.Len(t, history, 2, "WithHistory(2) must cap nested-apply chain at 2 items")
}

func Test_history_expect_failed_and_ignored_in_same_item(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	type instance = InstanceFSM[int, any]

	handlerExpected := func(_ context.Context, _ instance, _ ...any) error {
		return nil
	}

	handlerActual := func(_ context.Context, inst instance, _ ...any) error {
		inst.IgnoreCurrentTransition()
		return nil
	}

	machine, _ := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(handlerActual),
		},
	}, WithFullHistory[any]())

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerExpected)).
		Apply(t.Context(), open))

	require.Equal(t, closed, machine.Current(), "Ignored should roll back state")

	history := machine.History()
	require.Len(t, history, 1)
	require.True(t, history[0].Ignored, "IgnoreCurrentTransition must set Ignored")
	require.True(t, history[0].ExpectFailed, "expectation mismatch must set ExpectFailed")
	require.True(t, history[0].HasViolation())
}
