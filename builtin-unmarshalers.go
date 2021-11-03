package args

import (
	"reflect"
	"time"
)

var builtinUnmarshallers = map[reflect.Type]func(s string, value interface{}) error{
	reflect.TypeOf((*time.Duration)(nil)): func(s string, value interface{}) error {
		d, err := time.ParseDuration(s)
		if err == nil {
			reflect.ValueOf(value).Elem().Set(reflect.ValueOf(d))
		}
		return err
	},
}
