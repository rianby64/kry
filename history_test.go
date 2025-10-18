package kry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_history_size_limit_to_3(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](2)

	if err := hk.Push("action1", 0, 1, nil, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory1, hk.Items())

	if err := hk.Push("action2", 1, 2, nil, "param2"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory2 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Error:  nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory2, hk.Items())

	if err := hk.Push("action3", 2, 3, nil, "param3"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory3 := []HistoryItem[string, int, string]{
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Error:  nil,
		},
		{
			Action: "action3",
			From:   2,
			To:     3,
			Params: []string{"param3"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory3, hk.Items())
}

func Test_history_no_size_limit(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](fullHistorySize)

	if err := hk.Push("action1", 0, 1, nil, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory1, hk.Items())

	if err := hk.Push("action2", 1, 2, nil, "param2"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory2 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Error:  nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory2, hk.Items())

	if err := hk.Push("action3", 2, 3, nil, "param3"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 3 {
		t.Fatalf("expected history count to be 3, got %d", hk.length)
	}

	expectedHistory3 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Error:  nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Error:  nil,
		},
		{
			Action: "action3",
			From:   2,
			To:     3,
			Params: []string{"param3"},
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory3, hk.Items())
}

func Test_history_in_machine(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_limited(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := New(close, []Transition[string, int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	}, WithHistory(1))

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, any]{
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_error_from_enter(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	machine, _ := New(close, []Transition[string, int, string]{
		{
			Name: "roger",
			Src:  []int{close},
			Dst:  roger,
		},
		{
			Name: "open",
			Src:  []int{roger},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				if param == "fail" {
					return ErrNotAllowed
				}

				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open, "fail"), ErrNotAllowed)
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"fail"},
			Error:  ErrNotAllowed,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_incorrect_transition_error(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	machine, _ := New(close, []Transition[string, int, string]{
		{
			Name: "roger",
			Src:  []int{close},
			Dst:  roger,
		},
		{
			Name: "open",
			Src:  []int{roger},
			Dst:  open,
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "roger", roger), ErrNotFound)
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Error:  ErrNotFound,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_force_state(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	machine, _ := New(close, []Transition[string, int, string]{
		{
			Name: "roger",
			Src:  []int{close},
			Dst:  roger,
		},
		{
			Name: "open",
			Src:  []int{roger},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				if param == "force-close" {
					return fsm.ForceState(close)
				}

				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open, "force-close"))
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "open",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"force-close"},
			Error:  nil,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_multiple_force_state(t *testing.T) {
	const (
		close int = iota
		roger
		open
	)

	machine, _ := New(close, []Transition[string, int, string]{
		{
			Name: "roger",
			Src:  []int{close},
			Dst:  roger,
		},
		{
			Name: "open",
			Src:  []int{roger},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				if param == "force-close-roger" {
					if err := fsm.ForceState(close); err != nil {
						return err
					}

					if err := fsm.ForceState(roger); err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open, "force-close-roger"))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "open",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   close,
			To:     roger,
			Params: nil,
			Error:  nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"force-close-roger"},
			Error:  nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Error:  nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Error:  nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}
