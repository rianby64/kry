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
	}

	errApply := machine.
		With(ExpectEnter(handlerClose)).
		Apply(context.TODO(), "open", open)

	require.NoError(t, errApply)
	require.Equal(t, open, machine.Current())
	require.Equal(t, expectedHistory, machine.History())
}
