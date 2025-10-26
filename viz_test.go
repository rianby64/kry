package kry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type vizSample1 struct {
}

func (v *vizSample1) Open(ctx context.Context, instance InstanceFSM[string, int, any], param any) error {
	return nil
}

func (v *vizSample1) Close(ctx context.Context, instance InstanceFSM[string, int, any], param any) error {
	return nil
}

func Test_visualization_case1(t *testing.T) {
	t.SkipNow()

	const (
		close int = iota
		open
	)

	expected := `	"0" -> "1" [ label = "enter=Open-fm" ];
	"1" -> "0" [ label = "enter=func1" ];
`

	handlers := &vizSample1{}
	anonymousFn := func(ctx context.Context, instance InstanceFSM[string, int, any], param any) error { return nil }

	transitions := []Transition[string, int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlers.Open,
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: anonymousFn,
		},
	}

	actual := VisualizeStateLinks(transitions)

	require.Equal(t, expected, actual)
}

func Test_visualization_case2(t *testing.T) {
	t.SkipNow()

	const (
		close int = iota
		open
	)

	expected := `subgraph cluster_0 {
	style=filled;
	color=lightgrey;
	node [style=filled,color=white];
	"0" -> "1" [ label = "enter=Open-fm" ];

	label = "open";
}

subgraph cluster_1 {
	style=filled;
	color=lightgrey;
	node [style=filled,color=white];
	"1" -> "0" [ label = "enter=Close-fm" ];

	label = "close";
}

`

	handlers := &vizSample1{}
	transitions := []Transition[string, int, any]{
		{
			Name:  "open",
			Src:   []int{close},
			Dst:   open,
			Enter: handlers.Open,
		},
		{
			Name:  "close",
			Src:   []int{open},
			Dst:   close,
			Enter: handlers.Close,
		},
	}

	actual := VisualizeActions(transitions)
	require.Equal(t, expected, actual)
}
