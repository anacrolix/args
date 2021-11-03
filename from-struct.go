package args

import (
	"fmt"
	"reflect"

	"github.com/huandu/xstrings"
)

func FromStruct(target interface{}) (params []Param) {
	value := reflect.ValueOf(target).Elem()
	type_ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		target := fieldValue.Addr().Interface()
		structField := type_.Field(i)
		argTag := structField.Tag.Get("arg")
		if argTag == "-" {
			continue
		}
		pm := &param{
			name:       fmt.Sprintf("%v.%v", type_.Name(), structField.Name),
			target:     target,
			positional: argTag == "positional",
			valid:      true,
			help:       structField.Tag.Get("help"),
		}
		arity := structField.Tag.Get("arity")
		switch target.(type) {
		case *bool, **bool:
			pm.nullary = true
			pm.parse = boolFlagParser(target)
			pm.satisfied = true
			pm.negative = "no"
		default:
			pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
				pm.satisfied = true
				switch arity {
				case "+", "*":
				default:
					pm.valid = false
				}
				err = unmarshalInto(args[0], target)
				if err != nil {
					err = fmt.Errorf("unmarshalling %q: %w", args[0], err)
				}
				return args[1:], err
			}
		}
		if !pm.positional {
			pm.long = []string{xstrings.ToKebabCase(structField.Name)}
		}
		switch arity {
		case "*", "?":
			pm.satisfied = true
		case "+", "":
		default:
			panic(fmt.Sprintf("unhandled arity %q on %v", arity, type_))
		}
		default_ := structField.Tag.Get("default")
		if default_ != "" {
			_, err := pm.parse([]string{default_}, false)
			if err != nil {
				panic(fmt.Errorf("setting default %q: %w", default_, err))
			}
		}
		params = append(params, pm)
	}
	return
}
