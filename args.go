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
	Target interface{}
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
		structField := type_.Field(i)
		params = append(params, &Option{
			Long:   structField.Name,
			Target: value.Field(i).Addr().Interface(),
		})
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
