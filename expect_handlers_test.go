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

func Test_option_expect_level2_enter_handler_case1_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[string, int, string]

	handlerOpen := func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.Apply(ctx, "roger", roger)
		}

		return nil
	}

	handlerRoger := func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	transitions := []Transition[string, int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlerOpen,
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: handlerRoger,
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: handlerClose,
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action:       "open",
			From:         close,
			To:           open,
			Params:       []string{"goto-roger"},
			ExpectFailed: true,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
		},
		{
			Action: "close",
			From:   roger,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerClose)).
		Apply(context.TODO(), "open", open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case2_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[string, int, string]

	var (
		handlerOpen,
		handlerRoger,
		handlerClose func(ctx context.Context, instance instance, param string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				With(ExpectEnter(handlerOpen)).
				Apply(ctx, "roger", roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	transitions := []Transition[string, int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlerOpen,
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: handlerRoger,
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: handlerClose,
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action:       "open",
			From:         close,
			To:           open,
			Params:       []string{"goto-roger"},
			ExpectFailed: true,
		},
		{
			Action:       "roger",
			From:         open,
			To:           roger,
			ExpectFailed: true,
		},
		{
			Action: "close",
			From:   roger,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerClose)).
		Apply(context.TODO(), "open", open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case3_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[string, int, string]

	var (
		handlerOpen,
		handlerRoger,
		handlerClose func(ctx context.Context, instance instance, param string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				With(ExpectEnter(handlerOpen)).
				Apply(ctx, "roger", roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	transitions := []Transition[string, int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlerOpen,
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: handlerRoger,
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: handlerClose,
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: []string{"goto-roger"},
		},
		{
			Action:       "roger",
			From:         open,
			To:           roger,
			ExpectFailed: true,
		},
		{
			Action: "close",
			From:   roger,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerOpen)).
		Apply(context.TODO(), "open", open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case4_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[string, int, string]

	var (
		handlerOpen,
		handlerRoger,
		handlerClose func(ctx context.Context, instance instance, param string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				Apply(ctx, "roger", roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param string) error {
		return nil
	}

	transitions := []Transition[string, int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlerOpen,
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: handlerRoger,
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: handlerClose,
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: []string{"goto-roger"},
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
		},
		{
			Action: "close",
			From:   roger,
			To:     close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerOpen)).
		Apply(context.TODO(), "open", open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}
