package kry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_history_size_limit_to_3(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](2, false)

	if err := hk.Push("action1", 0, 1, nil, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "action3",
			From:   2,
			To:     3,
			Params: []string{"param3"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory3, hk.Items())
}

func Test_history_no_size_limit(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](fullHistorySize, false)

	if err := hk.Push("action1", 0, 1, nil, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
		},
		{
			Action: "action3",
			From:   2,
			To:     3,
			Params: []string{"param3"},
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
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
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"fail"},
			Err:    ErrNotAllowed,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    ErrNotFound,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"force-close"},
			Err:    nil,
		},
		{
			Action: "open",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
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
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"force-close-roger"},
			Err:    nil,
		},
		{
			Action: "open",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_multiple_force_state_limited(t *testing.T) {
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

				return fsm.Apply(ctx, "roger-close", close, "force-close")
			},
		},
		{
			Name: "roger-close",
			Src:  []int{roger},
			Dst:  close,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				if param == "force-close" {
					if err := fsm.ForceState(close); err != nil {
						return err
					}
				}

				return nil
			},
		},
	}, WithFullHistory())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open, "force-close-roger"))
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"force-close-roger"},
			Err:    nil,
		},
		{
			Action: "open",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger-close",
			From:   roger,
			To:     close,
			Params: []string{"force-close"},
			Err:    nil,
		},
		{
			Action: "roger-close",
			From:   close,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_with_force_states(t *testing.T) {
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

	require.NoError(t, machine.ForceState(open))
	require.NoError(t, machine.ForceState(close))

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger))
	require.Equal(t, roger, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "open", open, "fail"), ErrNotAllowed)
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.ForceState(open))
	require.NoError(t, machine.ForceState(roger))

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "",
			From:   close,
			To:     open,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"fail"},
			Err:    ErrNotAllowed,
		},
		{
			Action: "roger",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
	}

	require.Equal(t, expectedHistory1, machine.History())

	require.NoError(t, machine.Apply(context.TODO(), "open", open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close))
	require.Equal(t, close, machine.Current())

	expectedHistory2 := []HistoryItem[string, int, string]{
		{
			Action: "",
			From:   close,
			To:     open,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: []string{"fail"},
			Err:    ErrNotAllowed,
		},
		{
			Action: "roger",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "roger",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    nil,
			Forced: true,
		},
		{
			Action: "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory2, machine.History())
}

func Test_history_no_size_limit_stacktrace(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](fullHistorySize, true)

	intentionalErr := fmt.Errorf("intentional error")
	if err := hk.Push("action1", 0, 1, intentionalErr, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action: "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    intentionalErr,
			Stack:  "... stack trace ...", // machine depends on runtime, so we just check it's not empty
			Reason: intentionalErr.Error(),
		},
	}
	history := hk.Items()
	require.Len(t, history, 1)
	item := history[0]
	require.Equal(t, expectedHistory1[0].Action, item.Action)
	require.Equal(t, expectedHistory1[0].From, item.From)
	require.Equal(t, expectedHistory1[0].To, item.To)
	require.Equal(t, expectedHistory1[0].Params, item.Params)
	require.Equal(t, expectedHistory1[0].Err, item.Err)
	require.NotEmpty(t, item.Stack)
	require.Equal(t, expectedHistory1[0].Reason, item.Reason)
}
