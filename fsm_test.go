package kry_test

import (
	"context"
	"errors"
	"testing"

	fsm "github.com/rianby64/kry"
	"github.com/stretchr/testify/require"
)

func Test_set_initialState_string_ok(t *testing.T) {
	machine, _ := fsm.New("INITIAL_STATE", []fsm.Transition[string, string, any]{})

	require.NotNil(t, machine)
	require.Equal(t, "INITIAL_STATE", machine.Current())
}

func Test_set_initialState_int_ok(t *testing.T) {
	machine, _ := fsm.New(1, []fsm.Transition[string, int, any]{})

	require.NotNil(t, machine)
	require.Equal(t, 1, machine.Current())
}

func Test_set_transitions_string_int_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())
}

func Test_set_transitions_force_state_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.ForceState(open))
	require.Equal(t, open, machine.Current())
}

func Test_set_transitions_force_incorrect_state_ok(t *testing.T) {
	const (
		close int = iota
		open
		incorrect
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.ForceState(incorrect), fsm.ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_event(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.Apply(context.TODO(), "incorrect", open), fsm.ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_state(t *testing.T) {
	const (
		close int = iota
		open
		initial
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{initial}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open), fsm.ErrNotFound)
	require.Equal(t, close, machine.Current())
}

func Test_execute_Enter_one_time_one_parameter(t *testing.T) {
	const (
		close int = iota
		open
	)

	type Param struct {
		Value string
	}

	var calledEnter bool

	machine, _ := fsm.New(
		close, // Initial state
		[]fsm.Transition[string, int, Param]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, Param], param Param) error {
					require.Equal(t, "test", param.Value)
					require.Equal(t, open, instance.Current())

					calledEnter = true
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, Param], param Param) error {
					t.Log("should not be called")
					t.FailNow()
					return nil
				},
			},
		},
	)

	require.Nil(t, machine.Apply(context.TODO(), "open", open, Param{Value: "test"}))
	require.Equal(t, open, machine.Current())
	require.True(t, calledEnter)
}

func Test_execute_force_state(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(
		close, // Initial state
		[]fsm.Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					require.Equal(t, open, instance.Current())
					require.NoError(t, instance.ForceState(close))
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					t.Log("should not be called")
					t.FailNow()
					return nil
				},
			},
		},
	)

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, close, machine.Current())
}

func Test_execute_event_case2(t *testing.T) {
	const (
		close int = iota
		open
	)

	enterOpenCalledTimes := 0
	enterCloseCalledTimes := 0

	machine, _ := fsm.New(
		close, // Initial state
		[]fsm.Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterCloseCalledTimes++
					return nil
				},
			},
		},
	)

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, 2, enterOpenCalledTimes)
	require.Equal(t, 1, enterCloseCalledTimes)
}

func Test_failed_enter_OK(t *testing.T) {
	const (
		close int = iota
		open
	)

	expectedError := errors.New("expected error")
	enterOpenCalledTimes := 0
	enterCloseCalledTimes := 0

	machine, _ := fsm.New(
		close, // Initial state
		[]fsm.Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterCloseCalledTimes++
					return expectedError
				},
			},
		},
	)

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "close", close), expectedError)
	require.Equal(t, open, machine.Current())

	require.Equal(t, 2, enterOpenCalledTimes)
	require.Equal(t, 1, enterCloseCalledTimes)
}

func Test_execute_different_variadics(t *testing.T) {
	const (
		close int = iota
		open
	)

	var (
		calledOpenEnterNoParams,
		calledOpenEnter,
		calledOpenEnterVariadic,
		calledCloseEnterNoParams,
		calledCloseEnter,
		calledCloseEnterVariadic int
	)

	machine, _ := fsm.New(
		close, // Initial state
		[]fsm.Transition[string, int, int]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int]) error {
					calledOpenEnterNoParams++
					return nil
				},
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int], param int) error {
					calledOpenEnter++
					require.Equal(t, 1, param)
					return nil
				},
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int], param ...int) error {
					calledOpenEnterVariadic++
					require.Equal(t, 2, len(param))
					require.Equal(t, 3, param[0])
					require.Equal(t, 4, param[1])
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int]) error {
					calledCloseEnterNoParams++
					return nil
				},
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int], param int) error {
					calledCloseEnter++
					require.Equal(t, 2, param)
					return nil
				},
				EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, int], param ...int) error {
					calledCloseEnterVariadic++
					require.Equal(t, 2, len(param))
					require.Equal(t, 5, param[0])
					require.Equal(t, 6, param[1])
					return nil
				},
			},
		},
	)

	require.Nil(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())
	require.Equal(t, 1, calledOpenEnterNoParams)

	require.Nil(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledCloseEnterNoParams)

	require.Nil(t, machine.Apply(context.TODO(), "open", open, 1))
	require.Equal(t, open, machine.Current())
	require.Equal(t, 1, calledOpenEnter)

	require.Nil(t, machine.Apply(context.TODO(), "close", close, 2))
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledCloseEnter)

	require.Nil(t, machine.Apply(context.TODO(), "open", open, 3, 4))
	require.Equal(t, open, machine.Current())
	require.Equal(t, 1, calledOpenEnterVariadic)

	require.Nil(t, machine.Apply(context.TODO(), "close", close, 5, 6))
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledCloseEnterVariadic)
}

func Test_set_state_undefined_case1(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.Apply(context.TODO(), "open", 1)
	require.NoError(t, err)
	require.Equal(t, open, machine.Current())
}

func Test_set_state_undefined_case2(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(context.TODO(), "open", 1, 2))
	require.Equal(t, open, machine.Current())
}

func Test_set_state_undefined_case3(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterVariadic: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
				require.Equal(t, 1, param[0])
				return nil
			},
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	})

	require.NoError(t, machine.Apply(context.TODO(), "open", open, 1))
	require.Equal(t, open, machine.Current())
}

func Test_set_transitions_retrigger_ok(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	var (
		calledOpen,
		calledReopen,
		calledClose int
	)

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close, open}, Dst: roger,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				calledOpen++
				return nil
			},
		},
		{
			Name: "open", Src: []int{roger}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				calledReopen++
				return nil
			},
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				calledClose++
				return nil
			},
		},
	})

	require.ErrorIs(t, machine.Event(context.TODO(), "open"), fsm.ErrNotAllowed)

	require.NoError(t, machine.Apply(context.TODO(), "open", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, 2, calledOpen)
	require.Equal(t, 1, calledReopen)
	require.Equal(t, 1, calledClose)
}

func Test_set_repeated_transitions_panic(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	machine, err := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close, open}, Dst: roger,
		},
		{
			Name: "open", Src: []int{close}, Dst: roger,
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
		},
	})

	require.Nil(t, machine)
	require.ErrorIs(t, err, fsm.ErrRepeated)
}

func Test_set_event_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, err := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Event(context.TODO(), "close"))
	require.Equal(t, close, machine.Current())
}

func Test_transite_incorrect_event_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	errExpected := errors.New("expected error")

	machine, err := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				return errExpected
			},
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Event(context.TODO(), "incorrect"), fsm.ErrUnknown)
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Event(context.TODO(), "close"), errExpected)
	require.Equal(t, open, machine.Current())
}

func Test_loop_case_1(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	calledOpen := 0
	calledRoger := 0
	calledClose := 0

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledOpen > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen++
				return instance.Apply(ctx, "roger", roger)
			},
		},
		{
			Name: "roger", Src: []int{open, close}, Dst: roger,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledRoger > 0 {
					t.Log("roger should not be called more than one time")
					t.FailNow()
				}
				calledRoger++
				return nil
			},
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledClose > 0 {
					t.Log("close should not be called more than one time")
					t.FailNow()
				}
				calledClose++
				return nil
			},
		},
	})

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, roger, machine.Current())
	require.Equal(t, 1, calledOpen)
	require.Equal(t, 1, calledRoger)
	require.Equal(t, 0, calledClose)
}

func Test_loop_case_infinity_break(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	calledOpen := 0
	calledRoger := 0
	calledClose := 0

	machine, _ := fsm.New(close, []fsm.Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledOpen > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen++
				return instance.Apply(ctx, "roger", roger)
			},
		},
		{
			Name: "roger", Src: []int{open, close}, Dst: roger,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledRoger > 0 {
					t.Log("roger should not be called more than one time")
					t.FailNow()
				}
				calledRoger++
				return instance.Apply(ctx, "close", close)
			},
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any]) error {
				if calledClose > 0 {
					t.Log("close should not be called more than one time")
					t.FailNow()
				}
				calledClose++
				return instance.Apply(ctx, "open", open) // here I introduced a loop intentionally
			},
		},
	})

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open), fsm.ErrLoopFound)
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledOpen)
	require.Equal(t, 1, calledRoger)
	require.Equal(t, 1, calledClose)
}
