package main

import (
	"context"
	"fmt"

	"github.com/rianby64/kry"
)

type CustomParam struct {
	Value string
}

func main() {
	const (
		initial int = iota
		close
		open
	)

	ctx := context.TODO()

	fsk, err := kry.New(initial, []kry.Transition[string, int, CustomParam]{
		{
			Name: "open",
			Src:  []int{initial, close},
			Dst:  open,
			Enter: func(ctx context.Context, instance kry.InstanceFSM[string, int, CustomParam], param CustomParam) error {
				fmt.Println("Opened with param:", param.Value)

				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, kry.WithFullHistory())
	if err != nil {
		panic(err)
	}

	if err := fsk.Event(ctx, "open", CustomParam{Value: "example"}); err != nil {
		panic(err)
	}

	fmt.Println("Current state:", fsk.Current())

	if err := fsk.Apply(ctx, "close", close); err != nil {
		panic(err)
	}

	fmt.Println("Current state:", fsk.Current())
}
