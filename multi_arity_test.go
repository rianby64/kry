package kry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Exact Src + Exact Dst (pathByExact) ──────────────────────────────────────
//
// Each (src, dst) pair accepts at most one handler — any second registration,
// regardless of arity, must return ErrRepeated at construction.

func Test_multi_arity_exact_same_OnEnter_errors(t *testing.T) {
	const (
		moving int = iota + 1
		stopped
	)

	h := func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }

	_, err := New(moving, []Transition[int, float64]{
		{Name: "stop", Src: []int{moving}, Dst: stopped, Enter: OnEnter(h)},
		{Name: "stop", Src: []int{moving}, Dst: stopped, Enter: OnEnter(h)},
	})

	require.ErrorIs(t, err, ErrRepeated)
}

func Test_multi_arity_exact_OnEnter_then_OnEnterVariadic_errors(t *testing.T) {
	const (
		moving int = iota + 1
		stopped
	)

	_, err := New(moving, []Transition[int, float64]{
		{
			Name:  "stop",
			Src:   []int{moving},
			Dst:   stopped,
			Enter: OnEnter(func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }),
		},
		{
			Name:  "stop",
			Src:   []int{moving},
			Dst:   stopped,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, float64], _ ...float64) error { return nil }),
		},
	})

	require.ErrorIs(t, err, ErrRepeated)
}

func Test_multi_arity_exact_OnEnterVariadic_then_OnEnter_errors(t *testing.T) {
	const (
		moving int = iota + 1
		stopped
	)

	_, err := New(moving, []Transition[int, float64]{
		{
			Name:  "stop",
			Src:   []int{moving},
			Dst:   stopped,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, float64], _ ...float64) error { return nil }),
		},
		{
			Name:  "stop",
			Src:   []int{moving},
			Dst:   stopped,
			Enter: OnEnter(func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }),
		},
	})

	require.ErrorIs(t, err, ErrRepeated)
}

// ── SrcFn + Exact Dst (pathByMatchSrc) ───────────────────────────────────────
//
// Fn-based transitions have no uniqueness constraint — multiple registrations
// for the same Dst are allowed. First-registered wins at dispatch time.

func Test_multi_arity_srcfn_multiple_registrations_ok(t *testing.T) {
	const (
		stopped int = iota + 1
		movingStraight
		movingLeft
		movingRight
	)

	isMoving := func(s int) bool { return s >= movingStraight && s <= movingRight }

	_, err := New(movingStraight, []Transition[int, float64]{
		{
			Name:  "stop",
			SrcFn: isMoving,
			Dst:   stopped,
			Enter: OnEnter(func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }),
		},
		{
			Name:  "stop",
			SrcFn: isMoving,
			Dst:   stopped,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, float64], _ ...float64) error { return nil }),
		},
	})

	require.NoError(t, err)
}

// ── Exact Src + DstFn (pathByMatchDst) ───────────────────────────────────────

func Test_multi_arity_dstfn_multiple_registrations_ok(t *testing.T) {
	const (
		softStop int = iota + 1
		hardStop
		movingStraight
	)

	isStop := func(s int) bool { return s >= softStop && s <= hardStop }

	_, err := New(movingStraight, []Transition[int, float64]{
		{
			Name:  "stop",
			Src:   []int{movingStraight},
			DstFn: isStop,
			Enter: OnEnter(func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }),
		},
		{
			Name:  "stop",
			Src:   []int{movingStraight},
			DstFn: isStop,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, float64], _ ...float64) error { return nil }),
		},
	})

	require.NoError(t, err)
}

// ── SrcFn + DstFn (pathMatch) ─────────────────────────────────────────────────

func Test_multi_arity_srcfn_dstfn_multiple_registrations_ok(t *testing.T) {
	const (
		softStop int = iota + 1
		hardStop
		movingStraight
		movingLeft
		movingRight
	)

	isMoving := func(s int) bool { return s >= movingStraight && s <= movingRight }
	isStop := func(s int) bool { return s >= softStop && s <= hardStop }

	_, err := New(movingStraight, []Transition[int, float64]{
		{
			Name:  "stop",
			SrcFn: isMoving,
			DstFn: isStop,
			Enter: OnEnter(func(_ context.Context, _ InstanceFSM[int, float64], _ float64) error { return nil }),
		},
		{
			Name:  "stop",
			SrcFn: isMoving,
			DstFn: isStop,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, float64], _ ...float64) error { return nil }),
		},
	})

	require.NoError(t, err)
}

// ── SrcFn dispatch routing by current state ───────────────────────────────────
//
// Three SrcFn transitions registered in order: fn1 (zoneA→zoneB),
// fn2 (zoneB→final), fn3 (zoneC→zoneA). The apply sequence exercises:
//   1st Apply → exact transition (initial → zoneC)
//   2nd Apply → 3rd SrcFn fn3 (zoneC → zoneA)
//   3rd Apply → 1st SrcFn fn1 (zoneA → zoneB)
//   4th Apply → 2nd SrcFn fn2 (zoneB → final)

func Test_multi_arity_srcfn_dispatch_routes_by_current_state(t *testing.T) {
	const (
		initial int = iota
		zoneA       // fn1 matches this source
		zoneB       // fn2 matches this source
		zoneC       // fn3 matches this source
		final
	)

	var exactCalled, fn1Called, fn2Called, fn3Called int

	machine, err := New(initial, []Transition[int, any]{
		// exact transitions
		{
			Name: "to-zoneB",
			Src:  []int{initial},
			Dst:  zoneB,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		{
			Name: "to-zoneC",
			Src:  []int{initial},
			Dst:  zoneC,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		// SrcFn transitions — registered as fn1 (1st), fn2 (2nd), fn3 (3rd)
		{
			Name:  "fn1",
			SrcFn: func(s int) bool { return s == zoneA },
			Dst:   zoneB,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn1Called++
				return nil
			}),
		},
		{
			Name:  "fn2",
			SrcFn: func(s int) bool { return s == zoneB },
			Dst:   final,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn2Called++
				return nil
			}),
		},
		{
			Name:  "fn3",
			SrcFn: func(s int) bool { return s == zoneC },
			Dst:   zoneA,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn3Called++
				return nil
			}),
		},
	})
	require.NoError(t, err)

	// 1st apply: exact transition (initial → zoneC)
	require.NoError(t, machine.Apply(t.Context(), zoneC))
	require.Equal(t, zoneC, machine.Current())
	require.Equal(t, 1, exactCalled)

	// 2nd apply: 3rd SrcFn — fn3 matches zoneC, transitions to zoneA
	require.NoError(t, machine.Apply(t.Context(), zoneA))
	require.Equal(t, zoneA, machine.Current())
	require.Equal(t, 1, fn3Called)

	// 3rd apply: 1st SrcFn — fn1 matches zoneA, transitions to zoneB
	require.NoError(t, machine.Apply(t.Context(), zoneB))
	require.Equal(t, zoneB, machine.Current())
	require.Equal(t, 1, fn1Called)

	// 4th apply: 2nd SrcFn — fn2 matches zoneB, transitions to final
	require.NoError(t, machine.Apply(t.Context(), final))
	require.Equal(t, final, machine.Current())
	require.Equal(t, 1, fn2Called)

	require.Equal(t, 1, exactCalled)
	require.Equal(t, 0, fn1Called+fn2Called+fn3Called-3)
}

// ── DstFn dispatch routing by requested destination ───────────────────────────
//
// Three DstFn transitions registered in order: fn1 (zoneA→zoneB via DstFn),
// fn2 (zoneB→final via DstFn), fn3 (zoneC→zoneA via DstFn). The apply sequence:
//   1st Apply → exact transition (initial → zoneC)
//   2nd Apply → 3rd DstFn fn3 (zoneC → zoneA)
//   3rd Apply → 1st DstFn fn1 (zoneA → zoneB)
//   4th Apply → 2nd DstFn fn2 (zoneB → final)

func Test_multi_arity_dstfn_dispatch_routes_by_requested_destination(t *testing.T) {
	const (
		initial int = iota
		zoneA       // fn1 Src; fn3 DstFn matches this destination
		zoneB       // fn2 Src; fn1 DstFn matches this destination
		zoneC       // fn3 Src (reached via exact from initial)
		final       // fn2 DstFn matches this destination
	)

	var exactCalled, fn1Called, fn2Called, fn3Called int

	machine, err := New(initial, []Transition[int, any]{
		// exact transitions
		{
			Name: "to-zoneB",
			Src:  []int{initial},
			Dst:  zoneB,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		{
			Name: "to-zoneC",
			Src:  []int{initial},
			Dst:  zoneC,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		// DstFn transitions — registered as fn1 (1st), fn2 (2nd), fn3 (3rd)
		{
			Name:  "fn1",
			Src:   []int{zoneA},
			DstFn: func(s int) bool { return s == zoneB },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn1Called++
				return nil
			}),
		},
		{
			Name:  "fn2",
			Src:   []int{zoneB},
			DstFn: func(s int) bool { return s == final },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn2Called++
				return nil
			}),
		},
		{
			Name:  "fn3",
			Src:   []int{zoneC},
			DstFn: func(s int) bool { return s == zoneA },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn3Called++
				return nil
			}),
		},
	})
	require.NoError(t, err)

	// 1st apply: exact transition (initial → zoneC)
	require.NoError(t, machine.Apply(t.Context(), zoneC))
	require.Equal(t, zoneC, machine.Current())
	require.Equal(t, 1, exactCalled)

	// 2nd apply: 3rd DstFn — fn3.DstFn matches zoneA, transitions to zoneA
	require.NoError(t, machine.Apply(t.Context(), zoneA))
	require.Equal(t, zoneA, machine.Current())
	require.Equal(t, 1, fn3Called)

	// 3rd apply: 1st DstFn — fn1.DstFn matches zoneB, transitions to zoneB
	require.NoError(t, machine.Apply(t.Context(), zoneB))
	require.Equal(t, zoneB, machine.Current())
	require.Equal(t, 1, fn1Called)

	// 4th apply: 2nd DstFn — fn2.DstFn matches final, transitions to final
	require.NoError(t, machine.Apply(t.Context(), final))
	require.Equal(t, final, machine.Current())
	require.Equal(t, 1, fn2Called)

	require.Equal(t, 1, exactCalled)
	require.Equal(t, 0, fn1Called+fn2Called+fn3Called-3)
}

// ── SrcFn+DstFn dispatch routing by both src and dst ─────────────────────────
//
// Three pathMatch transitions registered in order: fn1 (zoneA,zoneB),
// fn2 (zoneB,final), fn3 (zoneC,zoneA). The apply sequence:
//   1st Apply → exact transition (initial → zoneC)
//   2nd Apply → 3rd pathMatch fn3 (zoneC → zoneA)
//   3rd Apply → 1st pathMatch fn1 (zoneA → zoneB)
//   4th Apply → 2nd pathMatch fn2 (zoneB → final)

func Test_multi_arity_srcfn_dstfn_dispatch_routes_by_both(t *testing.T) {
	const (
		initial int = iota
		zoneA       // fn1 SrcFn matches this; fn3 DstFn matches this
		zoneB       // fn2 SrcFn matches this; fn1 DstFn matches this
		zoneC       // fn3 SrcFn matches this (reached via exact from initial)
		final       // fn2 DstFn matches this
	)

	var exactCalled, fn1Called, fn2Called, fn3Called int

	machine, err := New(initial, []Transition[int, any]{
		// exact transitions
		{
			Name: "to-zoneB",
			Src:  []int{initial},
			Dst:  zoneB,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		{
			Name: "to-zoneC",
			Src:  []int{initial},
			Dst:  zoneC,
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				exactCalled++
				return nil
			}),
		},
		// SrcFn+DstFn transitions — registered as fn1 (1st), fn2 (2nd), fn3 (3rd)
		{
			Name:  "fn1",
			SrcFn: func(s int) bool { return s == zoneA },
			DstFn: func(s int) bool { return s == zoneB },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn1Called++
				return nil
			}),
		},
		{
			Name:  "fn2",
			SrcFn: func(s int) bool { return s == zoneB },
			DstFn: func(s int) bool { return s == final },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn2Called++
				return nil
			}),
		},
		{
			Name:  "fn3",
			SrcFn: func(s int) bool { return s == zoneC },
			DstFn: func(s int) bool { return s == zoneA },
			Enter: OnEnterVariadic(func(_ context.Context, _ InstanceFSM[int, any], _ ...any) error {
				fn3Called++
				return nil
			}),
		},
	})
	require.NoError(t, err)

	// 1st apply: exact transition (initial → zoneC)
	require.NoError(t, machine.Apply(t.Context(), zoneC))
	require.Equal(t, zoneC, machine.Current())
	require.Equal(t, 1, exactCalled)

	// 2nd apply: 3rd pathMatch — fn3.SrcFn matches zoneC, fn3.DstFn matches zoneA
	require.NoError(t, machine.Apply(t.Context(), zoneA))
	require.Equal(t, zoneA, machine.Current())
	require.Equal(t, 1, fn3Called)

	// 3rd apply: 1st pathMatch — fn1.SrcFn matches zoneA, fn1.DstFn matches zoneB
	require.NoError(t, machine.Apply(t.Context(), zoneB))
	require.Equal(t, zoneB, machine.Current())
	require.Equal(t, 1, fn1Called)

	// 4th apply: 2nd pathMatch — fn2.SrcFn matches zoneB, fn2.DstFn matches final
	require.NoError(t, machine.Apply(t.Context(), final))
	require.Equal(t, final, machine.Current())
	require.Equal(t, 1, fn2Called)

	require.Equal(t, 1, exactCalled)
	require.Equal(t, 0, fn1Called+fn2Called+fn3Called-3)
}
