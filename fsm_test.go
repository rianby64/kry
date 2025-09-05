package kry_test

import (
	"context"
	"errors"
	"testing"

	fsm "github.com/rianby64/kry"
	"github.com/stretchr/testify/require"
)

func Test_set_initialState_string_ok(t *testing.T) {
	machine := fsm.New("INITIAL_STATE", []fsm.Event[string, string, any]{})

	require.NotNil(t, machine)
	require.Equal(t, "INITIAL_STATE", machine.Current())
}

func Test_set_initialState_int_ok(t *testing.T) {
	machine := fsm.New(1, []fsm.Event[string, int, any]{})

	require.NotNil(t, machine)
	require.Equal(t, 1, machine.Current())
}

func Test_set_events_string_int_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine := fsm.New(close, []fsm.Event[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.Event(context.TODO(), "open")
	require.NoError(t, err)
	require.Equal(t, open, machine.Current())
}

func Test_set_events_force_state_ok(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine := fsm.New(close, []fsm.Event[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.ForceState(open)
	require.NoError(t, err)
	require.Equal(t, open, machine.Current())
}

func Test_set_events_force_incorrect_state_ok(t *testing.T) {
	const (
		close int = iota
		open
		incorrect
	)

	machine := fsm.New(close, []fsm.Event[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.ForceState(incorrect)
	require.ErrorIs(t, err, fsm.ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_event(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine := fsm.New(close, []fsm.Event[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.Event(context.TODO(), "incorrect")
	require.ErrorIs(t, err, fsm.ErrUnknown)
	require.Equal(t, close, machine.Current())
}

func Test_incorrect_state(t *testing.T) {
	const (
		close int = iota
		open
		initial
	)

	machine := fsm.New(close, []fsm.Event[string, int, any]{
		{Name: "open", Src: []int{initial}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	})

	err := machine.Event(context.TODO(), "open")
	require.ErrorIs(t, err, fsm.ErrNotFound)
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

	machine := fsm.New(
		close, // Initial state
		[]fsm.Event[string, int, Param]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, Param], param ...Param) error {
					require.Equal(t, 1, len(param), "expected one parameter")
					require.Equal(t, "test", param[0].Value)
					require.Equal(t, open, instance.Current())

					calledEnter = true
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, Param], param ...Param) error {
					t.Log("should not be called")
					t.FailNow()
					return nil
				},
			},
		},
	)

	err := machine.Event(context.TODO(), "open", Param{Value: "test"})
	require.Nil(t, err)
	require.Equal(t, open, machine.Current())
	require.True(t, calledEnter)
}

func Test_execute_force_state(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine := fsm.New(
		close, // Initial state
		[]fsm.Event[string, int, any]{
			{
				Name: "open",
				Src:  []int{close}, Dst: open,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					require.Equal(t, open, instance.Current())
					require.NoError(t, instance.ForceState(close))
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					t.Log("should not be called")
					t.FailNow()
					return nil
				},
			},
		},
	)

	err := machine.Event(context.TODO(), "open")
	require.Nil(t, err)
	require.Equal(t, close, machine.Current())
}

func Test_execute_event_case2(t *testing.T) {
	const (
		close int = iota
		open
	)

	enterOpenCalledTimes := 0
	enterCloseCalledTimes := 0

	machine := fsm.New(
		close, // Initial state
		[]fsm.Event[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterCloseCalledTimes++
					return nil
				},
			},
		},
	)

	require.Nil(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Event(context.TODO(), "close"))
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

	machine := fsm.New(
		close, // Initial state
		[]fsm.Event[string, int, any]{
			{
				Name: "open",
				Src:  []int{open, close}, Dst: open,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterOpenCalledTimes++
					return nil
				},
			},
			{
				Name: "close",
				Src:  []int{open}, Dst: close,
				Enter: func(ctx context.Context, instance fsm.InstanceFSM[string, int, any], param ...any) error {
					enterCloseCalledTimes++
					return expectedError
				},
			},
		},
	)

	require.Nil(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.Nil(t, machine.Event(context.TODO(), "open"))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Event(context.TODO(), "close"), expectedError)
	require.Equal(t, open, machine.Current())

	require.Equal(t, 2, enterOpenCalledTimes)
	require.Equal(t, 1, enterCloseCalledTimes)
}
