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
		targetType := reflect.TypeOf(pm.target)
		pm.valid = targetType.Kind() == reflect.Slice
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
	return pm
}
