<div style="text-align:center"><img src="https://github.com/rianby64/kry/blob/main/icon.png?raw=true" /></div>

# kry

A simple Go project for finite state machines (FSM). I got inspiration by my wife Kry. I really love her!

Thank you for using my code. Kry will be very happy if you like it.

## Overview

`kry` is a Go-based library for building and running finite state machines (FSM) in your applications. It provides a simple API to define states, transitions, and handle events.

## Getting Started

1. Install the package:

```sh
go get github.com/rianby64/kry
```

2. Import and use in your Go code:

```go
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
```

## Requirements

- Go 1.23 or higher

## We have

- Simple API
- Support for source transitions matching via function. (All 5xx, 4xx, etc.)
- Visualization tools for FSMs
- From transition - do a call to another transition, and do not allow looping
- History of transitions

## Wish list for future improvements

- Add more examples and documentation
- Implement more advanced features like state beforeEnter/exit actions. (should I?)
- Support for destination transitions matching via function. (All 5xx, 4xx, etc.)
- Make this lib safe for concurrent use

## Design considerations

1. Do not support `ForceState` - it is dangerous and breaks FSM concept.
  So, if you encounter such a need, please rethink your design.
2. Keep the API simple and easy to use.

## License

This project is licensed under the MIT License.
