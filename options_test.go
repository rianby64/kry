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
	})

	errApply := machine.
		With(ExpectEnter(handlerOpen)).
		Apply(context.TODO(), "open", open)

	// what if apply does not meet expectations? - oh yes! let's log it!
	// what if so?

	require.NoError(t, errApply)
	require.Equal(t, open, machine.Current())
}
