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
    "fmt"

    fsm "github.com/rianby64/kry"
)

func main() {
    const (
        close int = iota
        open
    )

    ctx := context.TODO()

    // Define states and transitions
    machine := fsm.New(
		close, // Initial state
		[]fsm.Event[string, int, any]{
			{
				Name:   "open",
				Source: []int{open, close}, Destination: open,
			},
			{
				Name:   "close",
				Source: []int{open}, Destination: close,
			},
		},
	)

    // Trigger events
    fmt.Println(machine.Current()) // Output: close
    machine.Event(ctx, "open")
    fmt.Println(machine.Current()) // Output: open
    machine.Event(ctx, "close")
    fmt.Println(machine.Current()) // Output: close
}
```

## Requirements

- Go 1.18 or higher

## License

This project is licensed under the MIT License.
