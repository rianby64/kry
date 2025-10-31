package kry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_panic_case1(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	ctx := context.Background()
	machine, _ := New(close, []Transition[string, int, any]{
		{
			Name: "open",
			Src:  []int{close},
			Dst:  open,
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
			EnterNoParams: func(ctx context.Context, instance InstanceFSM[string, int, any]) error {
				panic("intentional panic")
			},
		},
	},
		WithPanicHandler[any](func(ctx context.Context, panicReason any) {
			assert.Equal(t, "intentional panic", fmt.Sprint(panicReason))
		}),
		WithFullHistory[any](),
		WithEnabledStackTrace[any](),
	)

	require.NoError(t, machine.Apply(ctx, "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(ctx, "close", close))
	require.Equal(t, open, machine.Current())

	history := machine.History()
	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
		},
	}

	for index, historyItem := range history {
		assert.Equal(t, expectedHistory[index].Action, historyItem.Action)
		assert.Equal(t, expectedHistory[index].From, historyItem.From)
		assert.Equal(t, expectedHistory[index].To, historyItem.To)
		assert.Equal(t, expectedHistory[index].Params, historyItem.Params)
	}

	require.Contains(t, history[1].StackTrace, "panic_test.go:30")
}
