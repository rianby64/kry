package kry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_history_size_limit_to_3(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](2, false)

	if err := hk.Push("action1", 0, 1, nil, 3, "param1"); err != nil {
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

	if err := hk.Push("action2", 1, 2, nil, 3, "param2"); err != nil {
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

	if err := hk.Push("action3", 2, 3, nil, 3, "param3"); err != nil {
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

	if err := hk.Push("action1", 0, 1, nil, 3, "param1"); err != nil {
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

	if err := hk.Push("action2", 1, 2, nil, 3, "param2"); err != nil {
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

	if err := hk.Push("action3", 2, 3, nil, 3, "param3"); err != nil {
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

func Test_history_no_size_limit_stacktrace(t *testing.T) {
	hk := newHistoryKeeper[string, int, string](fullHistorySize, true)

	intentionalErr := fmt.Errorf("intentional error")
	if err := hk.Push("action1", 0, 1, intentionalErr, 3, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[string, int, string]{
		{
			Action:     "action1",
			From:       0,
			To:         1,
			Params:     []string{"param1"},
			Err:        intentionalErr,
			StackTrace: "... stack trace ...", // machine depends on runtime, so we just check it's not empty
			Reason:     intentionalErr.Error(),
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
	require.NotEmpty(t, item.StackTrace)
	require.Equal(t, expectedHistory1[0].Reason, item.Reason)
}

func Test_history_in_machine_apply_within_apply_case1(t *testing.T) {
	const (
		close int = iota
		roger1
		roger2
		roger3
		roger4
		roger5
		open
	)

	machine, _ := New(close, []Transition[string, int, string]{
		{
			Name: "roger",
			Src: []int{
				close,
			},
			Dst: roger1,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger2, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
			},
			Dst: roger2,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger3, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
			},
			Dst: roger3,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger4, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
				roger3,
			},
			Dst: roger4,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger5, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
				roger3,
				roger4,
			},
			Dst: roger5,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return nil
			},
		},
		{
			Name: "open",
			Src:  []int{roger5},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	const emptyString = ""

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger1, emptyString))
	require.Equal(t, roger5, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open, emptyString))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "close", close, emptyString))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "roger",
			From:   close,
			To:     roger1,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "roger",
			From:   roger1,
			To:     roger2,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "roger",
			From:   roger2,
			To:     roger3,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "roger",
			From:   roger3,
			To:     roger4,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "roger",
			From:   roger4,
			To:     roger5,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "open",
			From:   roger5,
			To:     open,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: []string{emptyString},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_in_machine_apply_within_apply_case2(t *testing.T) {
	const (
		close int = iota
		roger1
		roger2
		roger3
		roger4
		roger5
		roger6
		open
	)

	machine, _ := New(open, []Transition[string, int, string]{
		{
			Name: "roger",
			Src: []int{
				close,
			},
			Dst: roger1,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger2, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
			},
			Dst: roger2,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger3, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
			},
			Dst: roger3,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger4, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
				roger3,
			},
			Dst: roger4,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return fsm.Apply(ctx, "roger", roger6, param)
			},
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
				roger3,
				roger4,
			},
			Dst: roger5,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return nil
			},
		},
		{
			Name: "open",
			Src:  []int{roger5},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	const emptyString = ""

	require.NoError(t, machine.Apply(context.TODO(), "close", close, emptyString))
	require.Equal(t, close, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "roger", roger1, emptyString), ErrNotFound)
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "roger", roger5, emptyString))
	require.Equal(t, roger5, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger1,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Action: "roger",
			From:   roger1,
			To:     roger2,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Action: "roger",
			From:   roger2,
			To:     roger3,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Action: "roger",
			From:   roger3,
			To:     roger4,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Action: "roger",
			From:   roger4,
			To:     roger6,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Action: "roger",
			From:   close,
			To:     roger5,
			Params: []string{emptyString},
			Err:    nil,
		},
	}

	history := machine.History()
	require.Len(t, history, len(expectedHistory))

	for index, item := range history {
		require.Equal(t, expectedHistory[index].Action, item.Action)
		require.Equal(t, expectedHistory[index].From, item.From)
		require.Equal(t, expectedHistory[index].To, item.To)
		require.Equal(t, expectedHistory[index].Params, item.Params)
		require.ErrorIs(t, item.Err, expectedHistory[index].Err)
	}
}

func Test_history_in_machine_apply_within_apply_case3(t *testing.T) {
	const (
		close int = iota
		open
	)

	machine, _ := New(open, []Transition[string, int, string]{
		{
			Name: "open",
			Src:  []int{close},
			Dst:  open,
			Enter: func(ctx context.Context, fsm InstanceFSM[string, int, string], param string) error {
				return nil
			},
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory())

	const emptyString = ""

	require.NoError(t, machine.Apply(context.TODO(), "close", close, emptyString))
	require.Equal(t, close, machine.Current())

	require.ErrorIs(t, machine.Apply(context.TODO(), "unknown", open, emptyString), ErrUnknown)
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(context.TODO(), "open", open, emptyString))
	require.Equal(t, open, machine.Current())

	expectedHistory := []HistoryItem[string, int, string]{
		{
			Action: "close",
			From:   open,
			To:     close,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Action: "unknown",
			From:   close,
			To:     open,
			Params: []string{emptyString},
			Err:    ErrUnknown,
		},
		{
			Action: "open",
			From:   close,
			To:     open,
			Params: []string{emptyString},
			Err:    nil,
		},
	}

	history := machine.History()
	require.Len(t, history, len(expectedHistory))

	for index, item := range history {
		require.Equal(t, expectedHistory[index].Action, item.Action)
		require.Equal(t, expectedHistory[index].From, item.From)
		require.Equal(t, expectedHistory[index].To, item.To)
		require.Equal(t, expectedHistory[index].Params, item.Params)
		require.ErrorIs(t, item.Err, expectedHistory[index].Err)
	}
}
