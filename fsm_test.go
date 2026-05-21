package kry

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_set_initialState_string_ok(t *testing.T) {
	machine, _ := New("INITIAL_STATE", []Transition[string, any]{})

	require.NotNil(t, machine)
	require.Equal(t, "INITIAL_STATE", machine.Current())
}

func Test_previous_on_fresh_machine_returns_initial_state(t *testing.T) {
	const initialState = "INITIAL_STATE"

	machine, _ := New(initialState, []Transition[string, any]{})

	require.Equal(t, initialState, machine.Previous())
	require.Equal(t, initialState, machine.Previous())
}

func Test_previous_unchanged_after_failed_apply(t *testing.T) {
	const (
		close int = iota + 1
		open
		other
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, close, machine.Previous())

	require.ErrorIs(t, machine.Apply(t.Context(), other), ErrNotFound)
	require.Equal(t, close, machine.Previous())
}

func Test_set_initialState_int_ok(t *testing.T) {
	machine, _ := New(1, []Transition[int, any]{})

	require.NotNil(t, machine)
	require.Equal(t, 1, machine.Current())
}

func Test_undefined_src_state(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, err := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Dst: close},
	})

	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, machine)
}

func Test_set_transitions_int_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())
}

func Test_incorrect_event(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	// From close, there is no close→close transition
	require.ErrorIs(t, machine.Apply(t.Context(), close), ErrNotFound)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_state(t *testing.T) {
	const (
		close int = iota + 1
		open
		initial
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{initial}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.ErrorIs(t, machine.Apply(t.Context(), open), ErrNotFound)
	require.Equal(t, close, machine.Current())
}

func Test_execute_Enter_one_time_one_parameter(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type Param struct {
		Value string
	}

	var calledEnter bool

	machine, _ := New(
		close,
		[]Transition[int, Param]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, Param], param Param) error {
					require.Equal(t, "test", param.Value)
					require.Equal(t, open, instance.Current())
					require.Equal(t, close, instance.Previous())

					calledEnter = true
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, Param], param Param) error {
					t.Log("should not be called")
					t.FailNow()
					return nil
				}),
			},
		},
	)

	require.Nil(t, machine.Apply(t.Context(), open, Param{Value: "test"}))
	require.Equal(t, open, machine.Current())
	require.True(t, calledEnter)
}

func Test_execute_event_case2(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	enterOpenCalledTimes := 0
	enterCloseCalledTimes := 0

	machine, _ := New(
		close,
		[]Transition[int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
					enterCloseCalledTimes++
					return nil
				}),
			},
		},
	)

	require.Nil(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, 2, enterOpenCalledTimes)
	require.Equal(t, 1, enterCloseCalledTimes)
}

func Test_failed_enter_OK(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	expectedError := errors.New("expected error")
	enterOpenCalledTimes := 0
	enterCloseCalledTimes := 0

	machine, _ := New(
		close,
		[]Transition[int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
					enterCloseCalledTimes++
					return expectedError
				}),
			},
		},
	)

	require.Nil(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Apply(t.Context(), close), expectedError)
	require.Equal(t, open, machine.Current())

	require.Equal(t, 2, enterOpenCalledTimes)
	require.Equal(t, 1, enterCloseCalledTimes)
}

func Test_execute_different_variadics(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	t.Run("OnEnterVariadic", func(t *testing.T) {
		var calledOpen, calledClose int

		machine, _ := New(close, []Transition[int, int]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, int], param ...int) error {
					calledOpen++
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, int], param ...int) error {
					calledClose++
					return nil
				}),
			},
		})

		require.Nil(t, machine.Apply(t.Context(), open))
		require.Equal(t, open, machine.Current())
		require.Equal(t, 1, calledOpen)

		require.Nil(t, machine.Apply(t.Context(), close))
		require.Equal(t, close, machine.Current())
		require.Equal(t, 1, calledClose)
	})

	t.Run("OnEnter", func(t *testing.T) {
		var calledOpen, calledClose int

		machine, _ := New(close, []Transition[int, int]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, int], param int) error {
					require.Equal(t, 1, param)
					calledOpen++
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, int], param int) error {
					require.Equal(t, 2, param)
					calledClose++
					return nil
				}),
			},
		})

		require.Nil(t, machine.Apply(t.Context(), open, 1))
		require.Equal(t, open, machine.Current())
		require.Equal(t, 1, calledOpen)

		require.Nil(t, machine.Apply(t.Context(), close, 2))
		require.Equal(t, close, machine.Current())
		require.Equal(t, 1, calledClose)
	})

	t.Run("OnEnterVariadicMix", func(t *testing.T) {
		var calledOpen, calledClose int

		machine, _ := New(close, []Transition[int, int]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, int], param ...int) error {
					require.Equal(t, 2, len(param))
					require.Equal(t, 3, param[0])
					require.Equal(t, 4, param[1])
					calledOpen++
					return nil
				}),
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, int], param int) error {
					require.Equal(t, 6, param)
					calledClose++
					return nil
				}),
			},
		})

		require.Nil(t, machine.Apply(t.Context(), open, 3, 4))
		require.Equal(t, open, machine.Current())
		require.Equal(t, 1, calledOpen)

		require.Nil(t, machine.Apply(t.Context(), close, 6))
		require.Equal(t, close, machine.Current())
		require.Equal(t, 1, calledClose)
	})
}

func Test_set_state_undefined_case1(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())
}

func Test_set_state_undefined_case2(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	require.NoError(t, machine.Apply(t.Context(), open, 20))
	require.Equal(t, open, machine.Current())
}

func Test_set_state_undefined_case3(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				require.Equal(t, 1, param[0])
				return nil
			}),
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	})

	require.NoError(t, machine.Apply(t.Context(), open, 1))
	require.Equal(t, open, machine.Current())
}

func Test_set_transitions_retrigger_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	var (
		calledOpen,
		calledReopen,
		calledClose int
	)

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close, open}, Dst: roger,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledOpen++
				return nil
			}),
		},
		{
			Name: "open", Src: []int{roger}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledReopen++
				return nil
			}),
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledClose++
				return nil
			}),
		},
	})

	// From close, the only valid destination is roger; open has no close→open transition
	require.ErrorIs(t, machine.Apply(t.Context(), open), ErrNotFound)

	require.NoError(t, machine.Apply(t.Context(), roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, 2, calledOpen)
	require.Equal(t, 1, calledReopen)
	require.Equal(t, 1, calledClose)
}

func Test_set_repeated_transitions_error(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	machine, err := New(close, []Transition[int, any]{
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
		close int = iota + 1
		open
	)

	machine, err := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())
}

func Test_transite_incorrect_event_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	errExpected := errors.New("expected error")

	machine, err := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				return errExpected
			}),
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	// From open, there is no open→open transition
	require.ErrorIs(t, machine.Apply(t.Context(), open), ErrNotFound)
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Apply(t.Context(), close), errExpected)
	require.Equal(t, open, machine.Current())
}

func Test_loop_case_1(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	calledOpen := 0
	calledRoger := 0
	calledClose := 0

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledOpen > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen++
				return instance.Apply(ctx, roger)
			}),
		},
		{
			Name: "roger", Src: []int{open, close}, Dst: roger,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledRoger > 0 {
					t.Log("roger should not be called more than one time")
					t.FailNow()
				}
				calledRoger++
				return nil
			}),
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledClose > 0 {
					t.Log("close should not be called more than one time")
					t.FailNow()
				}
				calledClose++
				return nil
			}),
		},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, roger, machine.Current())
	require.Equal(t, 1, calledOpen)
	require.Equal(t, 1, calledRoger)
	require.Equal(t, 0, calledClose)
}

func Test_loop_case_infinity_break(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	calledOpen := 0
	calledRoger := 0
	calledClose := 0

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledOpen > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen++
				require.Equal(t, close, instance.Previous())

				return instance.Apply(ctx, roger)
			}),
		},
		{
			Name: "roger", Src: []int{open, close}, Dst: roger,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledRoger > 0 {
					t.Log("roger should not be called more than one time")
					t.FailNow()
				}
				calledRoger++
				require.Equal(t, open, instance.Previous())

				return instance.Apply(ctx, close)
			}),
		},
		{
			Name: "close", Src: []int{roger, open}, Dst: close,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledClose > 0 {
					t.Log("close should not be called more than one time")
					t.FailNow()
				}
				calledClose++
				return instance.Apply(ctx, open) // intentional loop
			}),
		},
	}, WithFullHistory[any]())

	require.ErrorIs(t, machine.Apply(t.Context(), open), ErrLoopFound)
	require.Equal(t, close, machine.Current())
	require.Equal(t, 1, calledOpen)
	require.Equal(t, 1, calledRoger)
	require.Equal(t, 1, calledClose)

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:   "open",
			From:   close,
			To:     open,
			Params: nil,
			Err:    ErrLoopFound,
		},
		{
			Name:   "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    ErrLoopFound,
		},
		{
			Name:   "close",
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
		require.Equal(t, expectedHistory[i].Name, item.Name, "Name at index %d", i)
		require.Equal(t, expectedHistory[i].From, item.From, "From at index %d", i)
		require.Equal(t, expectedHistory[i].To, item.To, "To at index %d", i)
		require.Equal(t, expectedHistory[i].Params, item.Params, "Params at index %d", i)
		require.ErrorIs(t, item.Err, expectedHistory[i].Err, "Error at index %d", i)
	}
}

func Test_loop_case_infinity_break_two_machines(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	var (
		machine1,
		machine2 *FSM[int, any]
	)

	calledOpen1 := 0
	calledOpen2 := 0

	machine1, _ = New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledOpen1 > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen1++
				return machine2.Apply(ctx, open)
			}),
		},
	})

	machine2, _ = New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				if calledOpen2 > 0 {
					t.Log("open should not be called more than one time")
					t.FailNow()
				}
				calledOpen2++
				return nil
			}),
		},
	})

	require.NoError(t, machine1.Apply(t.Context(), open))
	require.Equal(t, open, machine1.Current())
	require.Equal(t, open, machine2.Current())
	require.Equal(t, 1, calledOpen1)
	require.Equal(t, 1, calledOpen2)
}

func Test_set_transitions_match_fn(t *testing.T) {
	const (
		close int = iota + 1
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

	ctx := t.Context()
	transitions := []Transition[int, any]{
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
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledRoger1 = true

				return nil
			}),
		},
		{
			Name: "roger-trap",
			SrcFn: func(state int) bool {
				return open1 <= state && state <= open3
			},
			Dst: roger3,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledRogerMatch = true

				return nil
			}),
		},

		{
			Name: "close",
			SrcFn: func(state int) bool {
				return roger1 <= state && state <= roger3
			},
			Dst: close,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				calledClosed = true

				return nil
			}),
		},
	}
	machine, errConstructor := New(close, transitions)
	require.NoError(t, errConstructor)

	require.NoError(t, machine.Apply(ctx, open3))
	require.Equal(t, open3, machine.Current())

	require.NoError(t, machine.Apply(ctx, roger3))
	require.Equal(t, roger3, machine.Current())
	require.True(t, calledRogerMatch)

	require.NoError(t, machine.Apply(ctx, close))
	require.Equal(t, close, machine.Current())
	require.True(t, calledClosed)

	require.NoError(t, machine.Apply(ctx, open1))
	require.Equal(t, open1, machine.Current())

	require.NoError(t, machine.Apply(ctx, roger1))
	require.Equal(t, roger1, machine.Current())
	require.True(t, calledRoger1)
}

func Test_set_transitions_match_fn_error(t *testing.T) {
	const (
		close int = iota + 1
		open1
		open2
		open3
		roger
	)

	expectedError := fmt.Errorf("expected error")
	ctx := t.Context()
	transitions := []Transition[int, any]{
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
			SrcFn: func(state int) bool {
				return open1 <= state && state <= open3
			},
			Dst: roger,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				return expectedError
			}),
		},

		{
			Name: "close",
			Src:  []int{roger},
			Dst:  close,
		},
	}
	machine, errConstructor := New(close, transitions)
	require.NoError(t, errConstructor)

	require.NoError(t, machine.Apply(ctx, open3))
	require.Equal(t, open3, machine.Current())

	require.ErrorIs(t, machine.Apply(ctx, roger), expectedError)
	require.Equal(t, open3, machine.Current())
}

func Test_ignore_transition_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, err := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				instance.IgnoreCurrentTransition()
				return nil
			}),
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	}, WithFullHistory[any]())

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, close, machine.Current())

	history := machine.History()
	require.True(t, history[0].Ignored)
}

func Test_ignore_transition_outside_skip(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, err := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
		},
		{
			Name: "close", Src: []int{open}, Dst: close,
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	machine.IgnoreCurrentTransition()

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())
}

func Test_force_state(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	machine, err := New(close, []Transition[int, any]{
		{
			Name: "open", Src: []int{close}, Dst: open,
			Enter: OnEnterVariadic(func(ctx context.Context, instance InstanceFSM[int, any], param ...any) error {
				return instance.ForceState(roger)
			}),
		},
		{
			Name: "close", Src: []int{open, roger}, Dst: close,
		},
	})

	require.NotNil(t, machine)
	require.NoError(t, err)

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())
}

func Test_transit_match_dst_case1(t *testing.T) {
	const (
		close int = iota + 1
		roger1
		roger2
		roger3
		roger4
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open",
			Src:  []int{close},
			Dst:  open,
		},
		{
			Name: "roger-trap",
			Src:  []int{open},
			DstFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
		},
		{
			Name: "close",
			SrcFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
			Dst: close,
		},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger3))
	require.Equal(t, roger3, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger4))
	require.Equal(t, roger4, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())
}

func Test_transit_match_dst_case2(t *testing.T) {
	const (
		close int = iota + 1
		roger1
		roger2
		roger3
		roger4
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{
			Name: "open",
			Src:  []int{close},
			Dst:  open,
		},
		{
			Name: "roger-trap",
			Src:  []int{open},
			DstFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
		},
		{
			Name: "roger-trap",
			SrcFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
			DstFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
		},
		{
			Name: "close",
			SrcFn: func(state int) bool {
				return roger1 <= state && state <= roger4
			},
			Dst: close,
		},
	})

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger3))
	require.Equal(t, roger3, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger1))
	require.Equal(t, roger1, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), roger4))
	require.Equal(t, roger4, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())
}

func Test_strict_arity_OnEnter(t *testing.T) {
	const (
		moving int = iota + 1
		stopped
	)

	var called bool

	machine, _ := New(moving, []Transition[int, int]{
		{
			Name: "stop",
			Src:  []int{moving},
			Dst:  stopped,
			Enter: OnEnter(func(ctx context.Context, instance InstanceFSM[int, int], param int) error {
				called = true
				return nil
			}),
		},
	})

	require.ErrorIs(t, machine.Apply(t.Context(), stopped), ErrNotAllowed)
	require.False(t, called)
	require.Equal(t, moving, machine.Current())

	require.ErrorIs(t, machine.Apply(t.Context(), stopped, 1, 2), ErrNotAllowed)
	require.False(t, called)
	require.Equal(t, moving, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), stopped, 42))
	require.True(t, called)
	require.Equal(t, stopped, machine.Current())
}
