package kry

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_set_initialState_string_ok(t *testing.T) {
	machine, _ := New("INITIAL_STATE", []Transition[string, string, any]{})

	require.NotNil(t, machine)
	require.Equal(t, "INITIAL_STATE", machine.Current())
}

func Test_set_initialState_int_ok(t *testing.T) {
	machine, _ := New(1, []Transition[string, int, any]{})

	require.NotNil(t, machine)
	require.Equal(t, 1, machine.Current())
}

func Test_undefined_src_state(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, err := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Dst: close},
	})

	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, machine)
}

func Test_set_transitions_string_int_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := New(close, []Transition[string, int, any]{
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

	machine, _ := New(close, []Transition[string, int, any]{
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

	machine, _ := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.ForceState(incorrect), ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_event(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.Apply(context.TODO(), "incorrect", open), ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_state(t *testing.T) {
	const (
		close int = iota
		open
		initial
	)

	machine, _ := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{initial}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open), ErrNotFound)
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

	machine, _ := New(
		close, // Initial state
		[]Transition[string, int, Param]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: func(ctx context.Context, instance InstanceFSM[string, int, Param], param Param) error {
					require.Equal(t, "test", param.Value)
					require.Equal(t, open, instance.Current())

					calledEnter = true
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance InstanceFSM[string, int, Param], param Param) error {
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

	machine, _ := New(
		close, // Initial state
		[]Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
					require.Equal(t, open, instance.Current())
					require.NoError(t, instance.ForceState(close))
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
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

	machine, _ := New(
		close, // Initial state
		[]Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
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

	machine, _ := New(
		close, // Initial state
		[]Transition[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
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

	machine, _ := New(
		close, // Initial state
		[]Transition[string, int, int]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, int]) error {
					calledOpenEnterNoParams++
					return nil
				},
				Enter: func(ctx context.Context, instance InstanceFSM[string, int, int], param int) error {
					calledOpenEnter++
					require.Equal(t, 1, param)
					return nil
				},
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, int], param ...int) error {
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
				EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, int]) error {
					calledCloseEnterNoParams++
					return nil
				},
				Enter: func(ctx context.Context, instance InstanceFSM[string, int, int], param int) error {
					calledCloseEnter++
					require.Equal(t, 2, param)
					return nil
				},
				EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, int], param ...int) error {
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

	machine, _ := New(close, []Transition[string, int, any]{
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

	machine, _ := New(close, []Transition[string, int, any]{
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

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterVariadic: func(ctx context.Context, instance InstanceFSM[string, int, any], param ...any) error {
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

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close, open}, Dst: roger,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledOpen++
				return nil
			},
		},
		{
			Name: "open", Src: []int{roger}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledReopen++
				return nil
			},
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledClose++
				return nil
			},
		},
	})

	require.ErrorIs(t, machine.Event(context.TODO(), "open"), ErrNotAllowed)

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

	machine, err := New(close, []Transition[string, int, any]{
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
	require.ErrorIs(t, err, ErrRepeated)
}

func Test_set_event_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, err := New(close, []Transition[string, int, any]{
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

	machine, err := New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				return errExpected
			},
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Event(context.TODO(), "incorrect"), ErrUnknown)
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

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
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
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
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
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
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

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
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
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
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
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				if calledClose > 0 {
					t.Log("close should not be called more than one time")
					t.FailNow()
				}
				calledClose++
				return instance.Apply(ctx, "open", open) // here I introduced a loop intentionally
			},
		},
	}, WithFullHistory())

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open), ErrLoopFound)
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledOpen)
	require.Equal(t, 1, calledRoger)
	require.Equal(t, 1, calledClose)

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: nil,
			Err:    ErrLoopFound,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    ErrLoopFound,
		},
		{
			Action: "close",
			From:   roger,
			To:     close,
			Params: nil,
			Err:    ErrLoopFound,
		},
	}

	if machine.History() == nil {
		t.Fatal("history is nil")
	}

	if len(machine.History()) != len(expectedHistory) {
		t.Fatalf("history length mismatch: got %d, want %d", len(machine.History()), len(expectedHistory))
	}

	for i, item := range machine.History() {
		require.Equal(t, expectedHistory[i].Action, item.Action, "Action at index %d", i)
		require.Equal(t, expectedHistory[i].From, item.From, "From at index %d", i)
		require.Equal(t, expectedHistory[i].To, item.To, "To at index %d", i)
		require.Equal(t, expectedHistory[i].Params, item.Params, "Params at index %d", i)
		require.ErrorIs(t, item.Err, expectedHistory[i].Err, "Error at index %d", i)
	}
}

func Test_loop_case_infinity_break_two_machines(t *testing.T) {
	const (
		close int = iota
		open
	)

	var (
		machine1,
		machine2 *FSM[string, int, any]
	)

	calledOpen1 := 0
	calledOpen2 := 0

	machine1, _ = New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				if calledOpen1 > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen1++
				return machine2.Apply(ctx, "open", open)
			},
		},
	})

	machine2, _ = New(close, []Transition[string, int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				if calledOpen2 > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen2++
				return nil
			},
		},
	})

	require.NoError(t, machine1.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine1.Current())
	require.Equal(t, open, machine2.Current())
	require.Equal(t, 1, calledOpen1)
	require.Equal(t, 1, calledOpen2)
}

func Test_set_transitions_match_fn(t *testing.T) {
	const (
		close int = iota
		open1
		open2
		open3
		roger1
		roger2
		roger3
	)

	calledClosed := false
	calledRoger1 := false
	calledRogerMatch := false

	ctx := context.TODO()
	transitions := []Transition[string, int, any]{
		{
			Name: "open-slightly",
			Src:  []int{close},
			Dst:  open1,
		},
		{
			Name: "open-normal",
			Src:  []int{close},
			Dst:  open2,
		},
		{
			Name: "open-full",
			Src:  []int{close},
			Dst:  open3,
		},

		{
			Name: "roger",
			Src:  []int{open1},
			Dst:  roger1,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledRoger1 = true

				return nil
			},
		},
		{
			Name: "roger-trap",
			Match: func(state int) bool {
				return open1 <= state && state <= open3
			},
			Dst: roger3,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledRogerMatch = true

				return nil
			},
		},

		{
			Name: "close",
			Match: func(state int) bool {
				return roger1 <= state && state <= roger3
			},
			Dst: close,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				calledClosed = true

				return nil
			},
		},
	}
	machine, errConstructor := New(close, transitions)
	require.NoError(t, errConstructor)

	require.NoError(t, machine.Apply(ctx, "open-full", open3))
	require.Equal(t, open3, machine.Current())

	require.NoError(t, machine.Apply(ctx, "roger-trap", roger3))
	require.Equal(t, roger3, machine.Current())
	require.True(t, calledRogerMatch)

	require.NoError(t, machine.Apply(ctx, "close", close))
	require.Equal(t, close, machine.Current())
	require.True(t, calledClosed)

	require.NoError(t, machine.Apply(ctx, "open-slightly", open1))
	require.Equal(t, open1, machine.Current())

	require.NoError(t, machine.Apply(ctx, "roger", roger1))
	require.Equal(t, roger1, machine.Current())
	require.True(t, calledRoger1)
}

func Test_set_transitions_match_fn_error(t *testing.T) {
	const (
		close int = iota
		open1
		open2
		open3
		roger
	)

	expectedError := fmt.Errorf("expected error")
	ctx := context.TODO()
	transitions := []Transition[string, int, any]{
		{
			Name: "open-slightly",
			Src:  []int{close},
			Dst:  open1,
		},
		{
			Name: "open-normal",
			Src:  []int{close},
			Dst:  open2,
		},
		{
			Name: "open-full",
			Src:  []int{close},
			Dst:  open3,
		},

		{
			Name: "roger-trap",
			Match: func(state int) bool {
				return open1 <= state && state <= open3
			},
			Dst: roger,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				return expectedError
			},
		},

		{
			Name: "close",
			Src:  []int{roger},
			Dst:  close,
		},
	}
	machine, errConstructor := New(close, transitions)
	require.NoError(t, errConstructor)

	require.NoError(t, machine.Apply(ctx, "open-full", open3))
	require.Equal(t, open3, machine.Current())

	require.ErrorIs(t, machine.Apply(ctx, "roger-trap", roger), expectedError)
	require.Equal(t, open3, machine.Current())
}
