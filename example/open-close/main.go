package main

import (
	"context"
	"fmt"

	"github.com/rianby64/kry"
)

type CustomParam struct {
	Value string
}

type State int

const (
	initial State = iota
	close
	open
)

func (s State) String() string {
	switch s {
	case initial:
		return "initial"
	case close:
		return "close"
	case open:
		return "open"
	default:
		return "unknown"
	}
}

func main() {

	ctx := context.TODO()

	fsk, err := kry.Build[State, CustomParam](initial).
		GoingTo(
			open,
			kry.From[State, CustomParam](initial, close).
				On(
					"open",
					kry.OnEnter(func(ctx context.Context, instance kry.InstanceFSM[State, CustomParam], param CustomParam) error {
						fmt.Println("Opened with param:", param.Value)
						return nil
					}),
				),
		).
		GoingTo(close,
			kry.From[State, CustomParam](open).On("close", kry.NoOp[State, CustomParam]()),
		).
		WithFullHistory().
		Create()
	if err != nil {
		panic(err)
	}

	if err := fsk.Apply(ctx, open, CustomParam{Value: "example"}); err != nil {
		panic(err)
	}

	fmt.Println("Current state:", fsk.Current())

	if err := fsk.Apply(ctx, close); err != nil {
		panic(err)
	}

	fmt.Println("Current state:", fsk.Current())
}
