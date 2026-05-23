package kry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_history_size_limit_to_3(t *testing.T) {
	hk := newHistoryKeeper[int, string](2, false, cloneHandler)

	if err := hk.Push("action1", 0, 1, nil, 3, false, false, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[int, string]{
		{
			Name:   "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory1, hk.Items())

	if err := hk.Push("action2", 1, 2, nil, 3, false, false, "param2"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory2 := []HistoryItem[int, string]{
		{
			Name:   "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
		},
		{
			Name:   "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory2, hk.Items())

	if err := hk.Push("action3", 2, 3, nil, 3, false, false, "param3"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory3 := []HistoryItem[int, string]{
		{
			Name:   "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
		},
		{
			Name:   "action3",
			From:   2,
			To:     3,
			Params: []string{"param3"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory3, hk.Items())
}

func Test_history_no_size_limit(t *testing.T) {
	hk := newHistoryKeeper[int, string](fullHistorySize, false, cloneHandler)

	if err := hk.Push("action1", 0, 1, nil, 3, false, false, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[int, string]{
		{
			Name:   "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory1, hk.Items())

	if err := hk.Push("action2", 1, 2, nil, 3, false, false, "param2"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 2 {
		t.Fatalf("expected history count to be 2, got %d", hk.length)
	}

	expectedHistory2 := []HistoryItem[int, string]{
		{
			Name:   "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
		},
		{
			Name:   "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory2, hk.Items())

	if err := hk.Push("action3", 2, 3, nil, 3, false, false, "param3"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	if hk.length != 3 {
		t.Fatalf("expected history count to be 3, got %d", hk.length)
	}

	expectedHistory3 := []HistoryItem[int, string]{
		{
			Name:   "action1",
			From:   0,
			To:     1,
			Params: []string{"param1"},
			Err:    nil,
		},
		{
			Name:   "action2",
			From:   1,
			To:     2,
			Params: []string{"param2"},
			Err:    nil,
		},
		{
			Name:   "action3",
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
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	}, WithFullHistory[any]())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:   "open",
			From:   close,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Name:   "close",
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
		close int = iota + 1
		open
	)

	machine, _ := New(close, []Transition[int, any]{
		{Name: "open", Src: []int{close}, Dst: open},
		{Name: "close", Src: []int{open}, Dst: close},
	}, WithHistory[any](1))

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[int, any]{
		{
			Name:   "close",
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
		close int = iota + 1
		roger
		open
	)

	machine, _ := New(close, []Transition[int, string]{
		{
			Name: "roger",
			Src:  []int{close},
			Dst:  roger,
		},
		{
			Name: "open",
			Src:  []int{roger},
			Dst:  open,
			Enter: OnEnterVariadic(func(ctx context.Context, fsm InstanceFSM[int, string], param ...string) error {
				if len(param) > 0 && param[0] == "fail" {
					return ErrNotAllowed
				}

				return nil
			}),
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory[string]())

	require.NoError(t, machine.Apply(t.Context(), roger))
	require.Equal(t, roger, machine.Current())

	require.ErrorIs(t, machine.Apply(t.Context(), open, "fail"), ErrNotAllowed)
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Name:   "open",
			From:   roger,
			To:     open,
			Params: []string{"fail"},
			Err:    ErrNotAllowed,
		},
		{
			Name:   "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Name:   "close",
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
		close int = iota + 1
		roger
		open
	)

	machine, _ := New(close, []Transition[int, string]{
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
	}, WithFullHistory[string]())

	require.NoError(t, machine.Apply(t.Context(), roger))
	require.Equal(t, roger, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, open, machine.Current())

	// No open→roger transition; Name will be empty in history
	require.ErrorIs(t, machine.Apply(t.Context(), roger), ErrNotFound)
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "roger",
			From:   close,
			To:     roger,
			Params: nil,
			Err:    nil,
		},
		{
			Name:   "open",
			From:   roger,
			To:     open,
			Params: nil,
			Err:    nil,
		},
		{
			Name:   "",
			From:   open,
			To:     roger,
			Params: nil,
			Err:    ErrNotFound,
		},
		{
			Name:   "close",
			From:   open,
			To:     close,
			Params: nil,
			Err:    nil,
		},
	}
	require.Equal(t, expectedHistory, machine.History())
}

func Test_history_no_size_limit_stacktrace(t *testing.T) {
	hk := newHistoryKeeper[int, string](fullHistorySize, true, cloneHandler)

	intentionalErr := fmt.Errorf("intentional error")
	if err := hk.Push("action1", 0, 1, intentionalErr, 3, false, false, "param1"); err != nil {
		t.Fatalf("failed to push history item: %v", err)
	}

	expectedHistory1 := []HistoryItem[int, string]{
		{
			Name:       "action1",
			From:       0,
			To:         1,
			Params:     []string{"param1"},
			Err:        intentionalErr,
			StackTrace: "... stack trace ...",
			Reason:     intentionalErr.Error(),
		},
	}
	history := hk.Items()
	require.Len(t, history, 1)
	item := history[0]
	require.Equal(t, expectedHistory1[0].Name, item.Name)
	require.Equal(t, expectedHistory1[0].From, item.From)
	require.Equal(t, expectedHistory1[0].To, item.To)
	require.Equal(t, expectedHistory1[0].Params, item.Params)
	require.Equal(t, expectedHistory1[0].Err, item.Err)
	require.NotEmpty(t, item.StackTrace)
	require.Equal(t, expectedHistory1[0].Reason, item.Reason)
}

func Test_history_in_machine_apply_within_apply_case1(t *testing.T) {
	const (
		close int = iota + 1
		roger1
		roger2
		roger3
		roger4
		roger5
		open
	)

	machine, _ := New(close, []Transition[int, string]{
		{
			Name: "roger",
			Src: []int{
				close,
			},
			Dst: roger1,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return fsm.Apply(ctx, roger2, param)
			}),
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
			},
			Dst: roger2,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return fsm.Apply(ctx, roger3, param)
			}),
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
			},
			Dst: roger3,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return fsm.Apply(ctx, roger4, param)
			}),
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
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return fsm.Apply(ctx, roger5, param)
			}),
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
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return nil
			}),
		},
		{
			Name: "open",
			Src:  []int{roger5},
			Dst:  open,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return nil
			}),
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory[string]())

	const emptyString = ""

	require.NoError(t, machine.Apply(t.Context(), roger1, emptyString))
	require.Equal(t, roger5, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open, emptyString))
	require.Equal(t, open, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), close, emptyString))
	require.Equal(t, close, machine.Current())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "roger",
			From:   close,
			To:     roger1,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "roger",
			From:   roger1,
			To:     roger2,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "roger",
			From:   roger2,
			To:     roger3,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "roger",
			From:   roger3,
			To:     roger4,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "roger",
			From:   roger4,
			To:     roger5,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "open",
			From:   roger5,
			To:     open,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "close",
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
		close int = iota + 1
		roger1
		roger2
		roger3
		roger4
		roger5
		roger6
		open
	)

	machine, _ := New(open, []Transition[int, string]{
		{
			Name: "roger",
			Src: []int{
				close,
			},
			Dst: roger1,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				require.Equal(t, close, fsm.Previous())

				return fsm.Apply(ctx, roger2, param)
			}),
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
			},
			Dst: roger2,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				require.Equal(t, roger1, fsm.Previous())

				return fsm.Apply(ctx, roger3, param)
			}),
		},
		{
			Name: "roger",
			Src: []int{
				close,
				roger1,
				roger2,
			},
			Dst: roger3,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				require.Equal(t, roger2, fsm.Previous())

				return fsm.Apply(ctx, roger4, param)
			}),
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
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				require.Equal(t, roger3, fsm.Previous())

				return fsm.Apply(ctx, roger6, param) // intentional dead-end: roger6 has no transition from roger4
			}),
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
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return nil
			}),
		},
		{
			Name: "open",
			Src:  []int{roger5},
			Dst:  open,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				require.Equal(t, roger5, fsm.Previous())

				return nil
			}),
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory[string]())

	const emptyString = ""

	require.NoError(t, machine.Apply(t.Context(), close, emptyString))
	require.Equal(t, close, machine.Current())

	require.ErrorIs(t, machine.Apply(t.Context(), roger1, emptyString), ErrNotFound)
	require.Equal(t, close, machine.Current())
	require.Equal(t, open, machine.Previous())

	require.NoError(t, machine.Apply(t.Context(), roger5, emptyString))
	require.Equal(t, roger5, machine.Current())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "close",
			From:   open,
			To:     close,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "roger",
			From:   close,
			To:     roger1,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "roger",
			From:   roger1,
			To:     roger2,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "roger",
			From:   roger2,
			To:     roger3,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "roger",
			From:   roger3,
			To:     roger4,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "",
			From:   roger4,
			To:     roger6,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "roger",
			From:   close,
			To:     roger5,
			Params: []string{emptyString},
			Err:    nil,
		},
	}

	history := machine.History()
	require.Len(t, history, len(expectedHistory))

	for index, item := range history {
		require.Equal(t, expectedHistory[index].Name, item.Name)
		require.Equal(t, expectedHistory[index].From, item.From)
		require.Equal(t, expectedHistory[index].To, item.To)
		require.Equal(t, expectedHistory[index].Params, item.Params)
		require.ErrorIs(t, item.Err, expectedHistory[index].Err)
	}
}

func Test_history_in_machine_apply_within_apply_case3(t *testing.T) {
	const (
		close int = iota + 1
		open
	)

	machine, _ := New(open, []Transition[int, string]{
		{
			Name: "open",
			Src:  []int{close},
			Dst:  open,
			Enter: OnEnter(func(ctx context.Context, fsm InstanceFSM[int, string], param string) error {
				return nil
			}),
		},
		{
			Name: "close",
			Src:  []int{open},
			Dst:  close,
		},
	}, WithFullHistory[string]())

	const emptyString = ""

	require.NoError(t, machine.Apply(t.Context(), close, emptyString))
	require.Equal(t, close, machine.Current())

	// No close→close transition; Name will be empty in history
	require.ErrorIs(t, machine.Apply(t.Context(), close, emptyString), ErrNotFound)
	require.Equal(t, close, machine.Current())

	require.NoError(t, machine.Apply(t.Context(), open, emptyString))
	require.Equal(t, open, machine.Current())

	expectedHistory := []HistoryItem[int, string]{
		{
			Name:   "close",
			From:   open,
			To:     close,
			Params: []string{emptyString},
			Err:    nil,
		},
		{
			Name:   "",
			From:   close,
			To:     close,
			Params: []string{emptyString},
			Err:    ErrNotFound,
		},
		{
			Name:   "open",
			From:   close,
			To:     open,
			Params: []string{emptyString},
			Err:    nil,
		},
	}

	history := machine.History()
	require.Len(t, history, len(expectedHistory))

	for index, item := range history {
		require.Equal(t, expectedHistory[index].Name, item.Name)
		require.Equal(t, expectedHistory[index].From, item.From)
		require.Equal(t, expectedHistory[index].To, item.To)
		require.Equal(t, expectedHistory[index].Params, item.Params)
		require.ErrorIs(t, item.Err, expectedHistory[index].Err)
	}
}

// ── ForceState history recording ─────────────────────────────────────────────

func Test_history_force_state_inside_apply(t *testing.T) {
	const (
		closed int = iota + 1
		roger
		open
	)

	machine, err := New(closed, []Transition[int, any]{
		{
			Name: "open", Src: []int{closed}, Dst: open,
			Enter: OnEnterVariadic(func(_ context.Context, instance InstanceFSM[int, any], _ ...any) error {
				return instance.ForceState(roger)
			}),
		},
		{Name: "close", Src: []int{open, roger}, Dst: closed},
	}, WithFullHistory[any]())

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))
	require.Equal(t, roger, machine.Current())

	history := machine.History()
	require.Len(t, history, 1)

	item := history[0]
	require.Equal(t, "open", item.Name)
	require.Equal(t, closed, item.From)
	require.Equal(t, open, item.To)         // intended destination
	require.True(t, item.Forced)
	require.NotNil(t, item.ForcedTo)
	require.Equal(t, roger, *item.ForcedTo) // actual destination
	require.True(t, item.HasViolation())
}

func Test_history_force_state_outside_apply(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
	}, WithFullHistory[any]())

	require.NoError(t, err)

	require.NoError(t, machine.ForceState(open))
	require.Equal(t, open, machine.Current())

	history := machine.History()
	require.Len(t, history, 1)

	item := history[0]
	require.Equal(t, "", item.Name)
	require.Equal(t, closed, item.From)
	require.Equal(t, open, item.To)
	require.True(t, item.Forced)
	require.NotNil(t, item.ForcedTo)
	require.Equal(t, open, *item.ForcedTo)
	require.True(t, item.HasViolation())
}

func Test_history_has_violation_false_for_normal_transition(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
	}, WithFullHistory[any]())

	require.NoError(t, err)
	require.NoError(t, machine.Apply(t.Context(), open))

	history := machine.History()
	require.Len(t, history, 1)
	require.False(t, history[0].HasViolation())
	require.False(t, history[0].Forced)
	require.Nil(t, history[0].ForcedTo)
}

func Test_history_force_state_outside_apply_no_recording_without_history_option(t *testing.T) {
	const (
		closed int = iota + 1
		open
	)

	machine, err := New(closed, []Transition[int, any]{
		{Name: "open", Src: []int{closed}, Dst: open},
	})

	require.NoError(t, err)
	require.NoError(t, machine.ForceState(open))
	require.Equal(t, open, machine.Current())
	require.Empty(t, machine.History())
}
