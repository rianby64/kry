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

	type instance = InstanceFSM[int, any]

	handlerOpen := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	machine, _ := New(close, []Transition[int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnterVariadic(handlerOpen),
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:         "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Name: "close",
			From: open,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_enter_no_params_handler_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type instance = InstanceFSM[int, any]

	handlerOpen := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	machine, _ := New(close, []Transition[int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnterVariadic(handlerOpen),
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:         "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Name: "close",
			From: open,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_enter_variadic_handler_ok(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	type instance = InstanceFSM[int, any]

	handlerOpen := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, params ...any) error {
		return nil
	}

	machine, _ := New(close, []Transition[int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnterVariadic(handlerOpen),
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}, WithFullHistory[any]())

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:         "open",
			From:         close,
			To:           open,
			ExpectFailed: true,
		},
		{
			Name: "close",
			From: open,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case1_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[int, string]

	handlerOpen := func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.Apply(ctx, roger)
		}

		return nil
	}

	handlerRoger := func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	handlerClose := func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	transitions := []Transition[int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnter(handlerOpen),
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: OnEnterVariadic(handlerRoger),
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:         "open",
			From:         close,
			To:           open,
			Params:       []string{"goto-roger"},
			ExpectFailed: true,
		},
		{
			Name: "roger",
			From: open,
			To:   roger,
		},
		{
			Name: "close",
			From: roger,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case2_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[int, string]

	var (
		handlerOpen  func(ctx context.Context, instance instance, param string) error
		handlerRoger func(ctx context.Context, instance instance, param ...string) error
		handlerClose func(ctx context.Context, instance instance, param ...string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				With(ExpectEnter(handlerOpen)).
				Apply(ctx, roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	transitions := []Transition[int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnter(handlerOpen),
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: OnEnterVariadic(handlerRoger),
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:         "open",
			From:         close,
			To:           open,
			Params:       []string{"goto-roger"},
			ExpectFailed: true,
		},
		{
			Name:         "roger",
			From:         open,
			To:           roger,
			ExpectFailed: true,
		},
		{
			Name: "close",
			From: roger,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnterVariadic(handlerClose)).
		Apply(t.Context(), open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case3_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[int, string]

	var (
		handlerOpen  func(ctx context.Context, instance instance, param string) error
		handlerRoger func(ctx context.Context, instance instance, param ...string) error
		handlerClose func(ctx context.Context, instance instance, param ...string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				With(ExpectEnter(handlerOpen)).
				Apply(ctx, roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	transitions := []Transition[int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnter(handlerOpen),
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: OnEnterVariadic(handlerRoger),
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "open",
			From:   close,
			To:     open,
			Params: []string{"goto-roger"},
		},
		{
			Name:         "roger",
			From:         open,
			To:           roger,
			ExpectFailed: true,
		},
		{
			Name: "close",
			From: roger,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerOpen)).
		Apply(t.Context(), open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}

func Test_option_expect_level2_enter_handler_case4_ok(t *testing.T) {
	const (
		close int = iota + 1
		roger
		open
	)

	type instance = InstanceFSM[int, string]

	var (
		handlerOpen  func(ctx context.Context, instance instance, param string) error
		handlerRoger func(ctx context.Context, instance instance, param ...string) error
		handlerClose func(ctx context.Context, instance instance, param ...string) error
	)

	handlerOpen = func(ctx context.Context, instance instance, param string) error {
		if param == "goto-roger" {
			return instance.
				Apply(ctx, roger)
		}

		return nil
	}

	handlerRoger = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	handlerClose = func(ctx context.Context, instance instance, param ...string) error {
		return nil
	}

	transitions := []Transition[int, string]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: OnEnter(handlerOpen),
		},
		{
			Name:  "roger",
			Src:   []int{open, close},
			Dst:   roger,
			Enter: OnEnterVariadic(handlerRoger),
		},
		{
			Name:  "close",
			Src:   []int{open, roger},
			Dst:   close,
			Enter: OnEnterVariadic(handlerClose),
		},
	}
	machine, _ := New(close, transitions, WithFullHistory[string]())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "open",
			From:   close,
			To:     open,
			Params: []string{"goto-roger"},
		},
		{
			Name: "roger",
			From: open,
			To:   roger,
		},
		{
			Name: "close",
			From: roger,
			To:   close,
		},
	}

	require.NoError(t, machine.
		With(ExpectEnter(handlerOpen)).
		Apply(t.Context(), open, "goto-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.
		Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	require.Equal(t, expectedHistory, machine.History())
}
