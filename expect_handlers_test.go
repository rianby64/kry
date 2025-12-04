package kry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_option_expect_enter_handler_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type instance = InstanceFSM[string, int, any]

	handlerOpen := func(ctx context.Context, instance instance, param any) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, param any) error {
		return nil
	}

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlerOpen,
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: handlerClose,
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action:       "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerClose)).
		Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnter(handlerClose)).
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_enter_no_params_handler_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type instance = InstanceFSM[string, int, any]

	handlerOpen := func(ctx context.Context, instance instance) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance) error {
		return nil
	}

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name:          "open",
			Src:           []int{close},
			Dst:           open,
			EnterNoParams: handlerOpen,
		},
		{
			Name:          "close",
			Src:           []int{open},
			Dst:           close,
			EnterNoParams: handlerClose,
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action:       "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterNoParams(handlerClose)).
		Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnterNoParams(handlerClose)).
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_enter_variadic_handler_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type instance = InstanceFSM[string, int, any]

	handlerOpen := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name:          "open",
			Src:           []int{close},
			Dst:           open,
			EnterVariadic: handlerOpen,
		},
		{
			Name:          "close",
			Src:           []int{open},
			Dst:           close,
			EnterVariadic: handlerClose,
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action:       "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}
