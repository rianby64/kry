package kry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Basic construction ────────────────────────────────────────────────────────

func Test_builder_initial_state_ok(t *testing.T) {
	machine, err := Build[string, any]("initial").Create()

	require.NoError(t, err)
	require.NotNil(t, machine)
	require.Equal(t, "initial", machine.Current())
}

// ── Exact Src + Exact Dst (GoingTo + From) ───────────────────────────────────

func Test_builder_exact_src_exact_dst_ok(t *testing.T) {
	const (
		closed = iota
		open
	)

	called := 0
	handler := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
		called++
		return nil
	})

	machine, err := Build[int, any](closed).
		GoingTo(open,
			From[int, any](closed).On("open", handler),
		).
		GoingTo(closed,
			From[int, any](open).On("close", handler),
		).
		Create()

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())
	require.Equal(t, closed, machine.Previous())
	require.NoError(t, machine.Apply(t.Context(), closed))
	require.Equal(t, closed, machine.Current())
	require.Equal(t, 2, called)
}

func Test_builder_exact_src_exact_dst_repeated_errors(t *testing.T) {
	const (
		closed = iota
		open
	)

	handler := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
		return nil
	})

	_, err := Build[int, any](closed).
		GoingTo(open,
			From[int, any](closed).On("open", handler),
		).
		GoingTo(open,
			From[int, any](closed).On("open-dup", handler),
		).
		Create()

	require.ErrorIs(t, err, ErrRepeated)
}

// ── FnSrc + Exact Dst (GoingTo + FnFrom) ─────────────────────────────────────

func Test_builder_fn_src_exact_dst_ok(t *testing.T) {
	const (
		s0 = iota
		s1
		s2
		s3
	)

	called := 0
	handler := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
		called++
		return nil
	})

	machine, err := Build[int, any](s0).
		GoingTo(s1,
			From[int, any](s0).On("toS1", handler),
		).
		GoingTo(s0,
			FnFrom[int, any](func(s int) bool { return s >= s1 && s <= s3 }).On("toS0", handler),
		).
		Create()

	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), s1))
	require.Equal(t, s1, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), s0))
	require.Equal(t, s0, machine.Current())

	require.Equal(t, 2, called)
}

// ── Exact Src + FnDst (FnGoingTo + From) ─────────────────────────────────────

func Test_builder_exact_src_fn_dst_ok(t *testing.T) {
	const (
		closed = iota
		roger1
		roger2
		roger3
	)

	called := 0
	handler := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
		called++
		return nil
	})

	machine, err := Build[int, any](closed).
		FnGoingTo(
			func(dst int) bool { return dst >= roger1 && dst <= roger3 },
			From[int, any](closed).On("open", handler),
		).
		GoingTo(closed,
			FnFrom[int, any](func(s int) bool { return s >= roger1 && s <= roger3 }).On("close", handler),
		).
		Create()

	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), roger2))
	require.Equal(t, roger2, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), closed))
	require.Equal(t, closed, machine.Current())

	require.Equal(t, 2, called)
}

// ── FnSrc + FnDst (FnGoingTo + FnFrom) ───────────────────────────────────────

func Test_builder_fn_src_fn_dst_ok(t *testing.T) {
	const (
		s0 = iota
		s1
		s2
		s3
	)

	called := 0
	handler := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
		called++
		return nil
	})

	machine, err := Build[int, any](s0).
		GoingTo(s1,
			From[int, any](s0).On("bootstrap", handler),
		).
		FnGoingTo(
			func(dst int) bool { return dst >= s2 && dst <= s3 },
			FnFrom[int, any](func(src int) bool { return src >= s1 && src <= s2 }).On("escalate", handler),
		).
		GoingTo(s0,
			FnFrom[int, any](func(s int) bool { return s >= s1 && s <= s3 }).On("reset", handler),
		).
		Create()

	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), s1))
	require.Equal(t, s1, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), s3))
	require.Equal(t, s3, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), s0))
	require.Equal(t, s0, machine.Current())

	require.Equal(t, 3, called)
}

// ── Options forwarding ────────────────────────────────────────────────────────

func Test_builder_with_full_history_ok(t *testing.T) {
	const (
		closed = iota
		open
	)

	machine, err := Build[int, any](closed).
		GoingTo(open,
			From[int, any](closed).On("open", OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				return nil
			})),
		).
		WithFullHistory().
		Create()

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))
	require.Len(t, machine.History(), 1)
}

// ── Transition without callback (no .On) ─────────────────────────────────────

func Test_builder_no_callback_ok(t *testing.T) {
	const (
		closed = iota
		open
	)

	// GoingTo with a SourceMatcher that has no .On entries produces no transitions;
	// verify the machine still has no error and the state is navigable via the raw
	// Transition fallback — actually, a source with no entries produces zero
	// transitions, so Apply should return ErrNotFound.
	machine, err := Build[int, any](closed).
		GoingTo(open,
			From[int, any](closed),
		).
		Create()

	require.NoError(t, err)
	// No entries were appended — no transition registered.
	require.ErrorIs(t, machine.Apply(t.Context(), open), ErrNotFound)
}

// ── Equivalence with New + []Transition ──────────────────────────────────────

func Test_builder_equivalent_to_new_api(t *testing.T) {
	const (
		closed = iota
		open
	)

	calls1, calls2 := 0, 0

	h1 := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error { calls1++; return nil })
	h2 := OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error { calls2++; return nil })

	m1, err1 := Build[int, any](closed).
		GoingTo(open, From[int, any](closed).On("open", h1)).
		GoingTo(closed, From[int, any](open).On("close", h1)).
		Create()

	m2, err2 := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open, Enter: h2},
		{Name: "close", Src: []int{open}, Dst: closed, Enter: h2},
	})

	require.NoError(t, err1)
	require.NoError(t, err2)

	ctx := t.Context()

	require.NoError(t, m1.Apply(ctx, open))
	require.NoError(t, m2.Apply(ctx, open))
	require.Equal(t, m1.Current(), m2.Current())

	require.NoError(t, m1.Apply(ctx, closed))
	require.NoError(t, m2.Apply(ctx, closed))
	require.Equal(t, m1.Current(), m2.Current())

	require.Equal(t, calls1, calls2)
}
