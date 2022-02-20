package args

import (
	"fmt"
	"reflect"
)

type Param interface {
	Usage() Usage
}

type paramParser func(args []string, negative bool) (unusedArgs []string, err error)

type param struct {
	target interface{}
	long   []string
	short  []rune
	run    func(SubCmdCtx) error
	// Doesn't take arguments (except any attached to a switch).
	nullary    bool
	parse      paramParser
	afterParse []func() error
	// The param is filled based on its position, rather than a switch
	positional bool
	// The param is still taking arguments
	valid     bool
	name      string
	help      string
	satisfied bool
	negative  string
	arity     arity
}

func (p *param) String() string {
	if p.name != "" {
		return p.name
	}
	return fmt.Sprint(p.long)
}

type Usage struct {
	Switches  []string
	Arguments []string
	Help      string
}

func (p *param) Usage() (u Usage) {
	if p.positional {
		u.Switches = append(p.long)
	} else {
		for _, l := range p.long {
			u.Switches = append(u.Switches, "--"+l)
		}
		for _, s := range p.short {
			u.Switches = append(u.Switches, "-"+string(s))
		}
	}
	if !p.nullary {
		u.Arguments = append(u.Arguments, reflect.TypeOf(p.target).Elem().String())
	}
	u.Help = p.help
	return
}

// This should be unnecessary with generics.
func (p *param) Bool() bool {
	v := reflect.ValueOf(p.target)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.IsValid() && v.Interface().(bool)
}

func (p *param) Value() interface{} {
	return reflect.ValueOf(p.target).Elem().Interface()
}
