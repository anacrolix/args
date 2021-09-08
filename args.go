package args

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
)

type FlagOpt struct {
	Long   string
	Target *bool
	Short  rune
}

var ExitSuccess = errors.New("exit success")

func HelpFlag() *flag {
	return &flag{
		long:  "help",
		short: 'h',
		run: func(p *Parser) error {
			p.PrintChoices(os.Stdout)
			return ExitSuccess
		},
	}
}

type pos struct {
	Name   string
	ok     bool
	Target interface{}
}

func (p *pos) Valid() bool {
	return !p.ok
}

func (p *pos) Parse(args []string) ([]string, error) {
	return args[1:], unmarshalInto(args[0], p.Target)
}

func unmarshalInto(s string, target interface{}) error {
	value := reflect.ValueOf(target).Elem()
	switch value.Type().Kind() {
	case reflect.String:
		value.SetString(s)
		return nil
	case reflect.Slice:
		x := reflect.New(value.Type().Elem())
		err := unmarshalInto(s, x.Interface())
		if err != nil {
			return fmt.Errorf("unmarshalling in to new element for %v: %w", value.Type(), err)
		}
		value.Set(reflect.Append(value, x.Elem()))
		return nil
	default:
		return fmt.Errorf("unhandled target type %v", value.Type())
	}
}

func (p pos) Usage() string {
	if p.Name != "" {
		return p.Name
	}
	return reflect.ValueOf(p.Target).Elem().Type().String()
}

func Pos(target interface{}) pos {
	return pos{
		Target: target,
	}
}

type Parser struct {
	args      *[]string
	flags     []*flag
	options   []*Option
	pos       []*pos
	subcmds   []subcommand
	RanSubCmd bool
	Err       error
}

func (p *Parser) Parse() error {
	for len(*p.args) > 0 {
		err := p.ParseOne()
		if err != nil {
			return err
		}
	}
	return nil
}

func catch(err *error) {
	if *err != nil {
		return
	}
	r := recover()
	if r == nil {
		return
	}
	rErr, ok := r.(error)
	if !ok {
		panic(r)
	}
	if rErr == ExitSuccess {
		*err = rErr
	}
}

func (p *Parser) ParseOne() (err error) {
	arg := (*p.args)[0]
	//if arg == "--" {
	//	return p.parsePositionalOnly(params...)
	//}
	if len(arg) > 2 && arg[:2] == "--" {
		matches := make([]*flag, 0, 1)
		for _, keyed := range p.flags {
			for _, l := range keyed.Long() {
				if l == arg[2:] {
					matches = append(matches, keyed)
				}
			}
		}
		switch len(matches) {
		case 0:
			return fmt.Errorf("unmatched switch %q", arg)
		case 1:
			*p.args = (*p.args)[1:]
			match := matches[0]
			if match.run != nil {
				p.RanSubCmd = true
				return match.run(p)
			}
			*p.args, err = match.Parse((*p.args)[:])
			return
		default:
			err = errors.New("matched multiple params")
			return
		}
	}
	for _, pos := range p.pos {
		if !pos.Valid() {
			continue
		}
		*p.args, err = pos.Parse(*p.args)
		return
	}
	for _, subcmd := range p.subcmds {
		if subcmd.Name != arg {
			continue
		}
		*p.args = (*p.args)[1:]
		err = subcmd.Run(SubCmdCtx{unusedArgs: p.args})
		if err != nil {
			err = fmt.Errorf("running subcommand %q: %w", subcmd.Name, err)
		}
		p.RanSubCmd = true
		return
	}
	return fmt.Errorf("unexpected argument: %q", arg)
}

func (p *Parser) eachChoice(each func(c Param)) {
	for _, choice := range p.flags {
		each(choice)
	}
	for _, choice := range p.options {
		each(choice)
	}
	for _, choice := range p.subcmds {
		each(choice)
	}
	for _, choice := range p.pos {
		each(choice)
	}
}

func (p *Parser) PrintChoices(w io.Writer) {
	p.eachChoice(func(c Param) {
		fmt.Fprintf(w, "  %v\n", c.Usage())
	})
}

type SubCmdCtx struct {
	unusedArgs *[]string
	err        error
}

func NewParser() *Parser {
	return &Parser{
		flags: []*flag{HelpFlag()},
	}
}

func (p *Parser) SetArgs(args *[]string) {
	p.args = args
}

func (me *SubCmdCtx) NewParser() *Parser {
	p := NewParser()
	p.SetArgs(me.unusedArgs)
	return p
}

type SubcommandRunner func(ctx SubCmdCtx) (err error)

type subcommand struct {
	Run  SubcommandRunner
	Name string
}

func (s subcommand) Usage() string {
	return s.Name
}

func Subcommand(name string, run SubcommandRunner) subcommand {
	return subcommand{Run: run, Name: name}
}

type Option struct {
	Long   string
	Target interface{}
}

func (o Option) Usage() string {
	return fmt.Sprintf("--%v", o.Long)
}

type Param interface {
	Usage() string
}

func FromStruct(target interface{}) (params []Param) {
	value := reflect.ValueOf(target).Elem()
	type_ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		target := fieldValue.Addr().Interface()
		structField := type_.Field(i)
		if structField.Tag.Get("arg") == "positional" {
			params = append(params, &pos{
				Name:   structField.Name,
				Target: target,
			})
		} else {
			params = append(params, &Option{
				Long:   structField.Name,
				Target: target,
			})
		}
	}
	return
}

func ParseMain(params ...Param) *Parser {
	return Parse(os.Args[1:], params...)
}

func (p *Parser) AddParams(params ...Param) *Parser {
	for _, param := range params {
		switch t := param.(type) {
		case *flag:
			p.flags = append(p.flags, t)
		case subcommand:
			p.subcmds = append(p.subcmds, t)
		case *Option:
			p.options = append(p.options, t)
		case *pos:
			p.pos = append(p.pos, t)
		default:
			panic(param)
		}
	}
	return p
}

func Parse(args []string, params ...Param) *Parser {
	p := Parser{
		args: &args,
	}
	p.AddParams(params...)
	p.Err = p.Parse()
	return &p
}

func FatalUsage() {
	os.Exit(2)
}
