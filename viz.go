package kry

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

var (
	nameOfFuncRegexp = regexp.MustCompile(`[.][^.\s]+$`)
)

func obtainFuncName(fn any) string {
	funcEnterName := ""

	if fn != nil {
		funcE := reflect.ValueOf(fn)
		funcPtr := funcE.Pointer()
		funcEnterName = runtime.FuncForPC(funcPtr).Name()
		nameOfFunc := nameOfFuncRegexp.FindString(funcEnterName)
		if len(nameOfFunc) > 0 && nameOfFunc[0] == '.' {
			funcEnterName = nameOfFunc[1:]
		}
	}

	return funcEnterName
}

func VisualizeStateLinks[State comparable, Param any](transitions []Transition[State, Param]) string {
	result := strings.Builder{}

	for _, transition := range transitions {
		var fn any
		switch transition.Enter.arity {
		case arityWith:
			fn = transition.Enter.with
		case arityVariadic:
			fn = transition.Enter.variadic
		}
		funcName := obtainFuncName(fn)

		for _, src := range transition.Src {
			label := ""
			if funcName != "" {
				label = fmt.Sprintf(` [ label = "enter=%s" ]`, funcName)
			}

			stateTransition := fmt.Sprintf(`%s"%v" -> "%v"%s;%s`, "\t", src, transition.Dst, label, "\n")
			result.WriteString(stateTransition)
		}
	}

	return result.String()
}

func VisualizeActions[State comparable, Param any](transitions []Transition[State, Param]) string {
	result := strings.Builder{}
	actionLinks := map[string][]string{}

	for _, transition := range transitions {
		links := VisualizeStateLinks([]Transition[State, Param]{transition})
		actionLinks[transition.Name] = append(actionLinks[transition.Name], links)
	}

	index := 0
	for actionName, links := range actionLinks {
		subgraph := fmt.Sprintf(`subgraph cluster_%d {
	style=filled;
	color=lightgrey;
	node [style=filled,color=white];
%s
	label = "%v";
}`, index, strings.Join(links, "\n"), actionName)

		result.WriteString(subgraph)
		result.WriteString("\n\n")
		index++
	}

	return result.String()
}
