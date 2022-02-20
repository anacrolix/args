package args

import (
	"fmt"
	"reflect"
)

func Pos(name string, target interface{}, opts ...ParamOpt) *param {
	pm := &param{
		target:     target,
		positional: true,
		valid:      true,
		name:       name,
	}
	pm.parse = func(args []string, negative bool) ([]string, error) {
		targetType := reflect.TypeOf(pm.target).Elem()
		switch pm.arity {
		case '+', '*':
		default:
			pm.valid = false
		}
		pm.satisfied = true
		err := unmarshalInto(args[0], pm.target)
		if err != nil {
			err = fmt.Errorf("unmarshalling %q into %v", args[0], targetType)
		}
		return args[1:], err
	}
	for _, opt := range opts {
		opt(pm)
	}
	switch pm.arity {
	case '*', '?':
		pm.satisfied = true
	}
	return pm
}
