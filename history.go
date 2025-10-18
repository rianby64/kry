package kry

import "github.com/fxamacker/cbor/v2"

const (
	fullHistorySize = -1
)

type HistoryItem[Action, State comparable, Param any] struct {
	Forced bool
	Action Action
	From   State
	To     State
	Params []Param
	Error  error
}

type historyItem[Action, State comparable, Param any] struct {
	*HistoryItem[Action, State, Param]
	Next *historyItem[Action, State, Param]
}

type historyKeeper[Action, State comparable, Param any] struct {
	maxLength int
	head      *historyItem[Action, State, Param]
	tail      *historyItem[Action, State, Param]
	length    int
}

func newHistoryKeeper[Action, State comparable, Param any](size int) *historyKeeper[Action, State, Param] {
	return &historyKeeper[Action, State, Param]{
		maxLength: size,
		head:      nil,
		tail:      nil,
		length:    0,
	}
}

func cloneParams[Param any](params ...Param) ([]Param, error) {
	if params == nil {
		return nil, nil
	}

	data, err := cbor.Marshal(params)
	if err != nil {
		return nil, err
	}

	var cloned []Param
	if err := cbor.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}

	return cloned, nil
}

func (hk *historyKeeper[Action, State, Param]) Push(action Action, from State, to State, err error, params ...Param) error {
	if hk.maxLength == 0 {
		return nil
	}

	cloneParams, errClone := cloneParams(params...)
	if errClone != nil {
		return errClone
	}

	item := &historyItem[Action, State, Param]{
		HistoryItem: &HistoryItem[Action, State, Param]{
			Action: action,
			From:   from,
			To:     to,
			Params: cloneParams,
			Error:  err,
		},
	}

	if hk.maxLength > 0 && hk.length >= hk.maxLength {
		hk.tail.Next = item
		hk.tail = item
		hk.head = hk.head.Next

		return nil
	}

	hk.length++
	if hk.head == nil {
		hk.head = item
		hk.tail = item
	} else {
		hk.tail.Next = item
		hk.tail = item
	}

	return nil
}

func (hk *historyKeeper[Action, State, Param]) Items() []HistoryItem[Action, State, Param] {
	items := make([]HistoryItem[Action, State, Param], hk.length)

	i := 0
	current := hk.head
	for current != nil {
		items[i] = *current.HistoryItem
		current = current.Next
		i++
	}

	return items
}

func (fsk *FSM[Action, State, Param]) History() []HistoryItem[Action, State, Param] {
	return fsk.historyKeeper.Items()
}
