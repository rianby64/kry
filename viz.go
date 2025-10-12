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

func VisualizeStateLinks[Action, State comparable, Param any](transitions []Transition[Action, State, Param]) string {
	result := strings.Builder{}

	for _, transition := range transitions {
		// Get the function name of transition.Enter using reflect
		funcEnterNoParamsName := obtainFuncName(transition.EnterNoParams)
		funcEnterName := obtainFuncName(transition.Enter)
		funcEnterVariadicName := obtainFuncName(transition.EnterVariadic)

		for _, src := range transition.Src {
			label := ""
			if funcEnterName != "" || funcEnterNoParamsName != "" || funcEnterVariadicName != "" {
				fns := []string{}
				if funcEnterNoParamsName != "" {
					fns = append(fns, fmt.Sprintf("enter0=%s", funcEnterNoParamsName))
				}
				if funcEnterName != "" {
					fns = append(fns, fmt.Sprintf("enter=%s", funcEnterName))
				}
				if funcEnterVariadicName != "" {
					fns = append(fns, fmt.Sprintf("enterV=%s", funcEnterVariadicName))
				}
				label = fmt.Sprintf(` [ label = "%s" ]`, strings.Join(fns, ", "))
			}

			stateTransition := fmt.Sprintf(`%s"%v" -> "%v"%s;%s`, "\t", src, transition.Dst, label, "\n")
			result.WriteString(stateTransition)
		}
	}

	return result.String()
}

func VisualizeActions[Action, State comparable, Param any](transitions []Transition[Action, State, Param]) string {
	result := strings.Builder{}
	actionLinks := map[Action][]string{} // action name to list of links

	for _, transition := range transitions {
		links := VisualizeStateLinks([]Transition[Action, State, Param]{transition})
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
