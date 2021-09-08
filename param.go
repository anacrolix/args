package args

import (
	"fmt"
	"reflect"
)

type Param interface {
	Usage() Usage
}

type param struct {
	target     interface{}
	long       []string
	short      []rune
	run        func(SubCmdCtx) error
	nullary    bool
	parse      func(args []string) (unusedArgs []string, err error)
	positional bool
	valid      bool
	name       string
	help       string
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
	for _, l := range p.long {
		u.Switches = append(u.Switches, "--"+l)
	}
	for _, s := range p.short {
		u.Switches = append(u.Switches, "-"+string(s))
	}
	if !p.nullary {
		u.Arguments = append(u.Arguments, p.name)
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
