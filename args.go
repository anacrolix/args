package args

import (
	"encoding"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
)

func HelpFlag() *param {
	return &param{
		long:  []string{"help"},
		short: []rune{'h'},
		run: func(p SubCmdCtx) error {
			p.parent.PrintChoices(os.Stdout)
			return ErrHelped
		},
		name:      "help",
		satisfied: true,
		nullary:   true,
		valid:     true,
	}
}

func unmarshalInto(s string, target interface{}) error {
	if herp, ok := target.(encoding.TextUnmarshaler); ok {
		return herp.UnmarshalText([]byte(s))
	}
	value := reflect.ValueOf(target).Elem()
	switch value.Type().Kind() {
	case reflect.String:
		value.SetString(s)
	case reflect.Slice:
		x := reflect.New(value.Type().Elem())
		err := unmarshalInto(s, x.Interface())
		if err != nil {
			return fmt.Errorf("unmarshalling in to new element for %v: %w", value.Type(), err)
		}
		value.Set(reflect.Append(value, x.Elem()))
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		value.SetBool(b)
	case reflect.Ptr:
		ptrNew := reflect.New(value.Type().Elem())
		err := unmarshalInto(s, ptrNew.Interface())
		if err != nil {
			return fmt.Errorf("unmarshalling into %v: %w", ptrNew, err)
		}
		value.Set(ptrNew)
	case reflect.Int64:
		i64, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return err
		}
		value.SetInt(i64)
	default:
		return fmt.Errorf("unhandled target type %v", value.Type())
	}
	return nil
}

type Main struct {
	Params     []Param
	AfterParse func() error
}

func (m Main) Do() {
	p := Parse(os.Args[1:], m.Params...)
	if p.Err != nil {
		if errors.Is(p.Err, ErrHelped) {
			return
		}
		log.Printf("error parsing args in main: %v", p.Err)
		FatalUsage()
	}
	if !p.RanSubCmd {
		p.Parser.PrintChoices(os.Stderr)
		FatalUsage()
	}
	if m.AfterParse != nil {
		m.AfterParse()
	}
	err := p.Run()
	if err != nil {
		log.Printf("error running main parse result: %v", err)
		FatalUsage()
	}
}

// Deprecated: Use Main
func ParseMain(params ...Param) {
	Main{
		Params: params,
	}.Do()
}

func Parse(args []string, params ...Param) (r ParseResult) {
	p := NewParser()
	p.SetArgs(&args)
	p.AddParams(params...)
	r.Parser = p
	r.Err = p.Parse()
	r.RanSubCmd = p.RanSubCmd
	return
}

func FatalUsage() {
	os.Exit(2)
}

type ParseResult struct {
	Err       error
	RanSubCmd bool
	Parser    *Parser
}

func (me ParseResult) Run() error {
	if me.Err != nil {
		return me.Err
	}
	for _, f := range me.Parser.deferred {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}
