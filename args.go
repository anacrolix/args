package args

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/huandu/xstrings"
)

type FlagOpt struct {
	Long   string
	Target *bool
	Short  rune
}

var ExitSuccess = errors.New("exit success")

func HelpFlag() *param {
	return &param{
		long:  []string{"help"},
		short: []rune{'h'},
		run: func(p SubCmdCtx) error {
			p.parent.PrintChoices(os.Stdout)
			return ExitSuccess
		},
		name:      "help",
		satisfied: true,
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
	default:
		return fmt.Errorf("unhandled target type %v", value.Type())
	}
	return nil
}

func Pos(name string, target interface{}) *param {
	pm := &param{
		target:     target,
		positional: true,
		valid:      true,
		name:       name,
	}
	pm.parse = func(args []string) ([]string, error) {
		targetType := reflect.TypeOf(pm.target)
		pm.valid = targetType.Kind() == reflect.Slice
		err := unmarshalInto(args[0], pm.target)
		if err != nil {
			err = fmt.Errorf("unmarshalling %q into %v", args[0], targetType)
		}
		return args[1:], err
	}
	return pm
}

type Parser struct {
	args      *[]string
	params    []*param
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
	for _, pm := range p.params {
		if !pm.satisfied {
			return fmt.Errorf("parameter not satisfied: %v", pm)
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

func filterParams(pms []*param, f func(pm *param) bool) (ret []*param) {
	for _, pm := range pms {
		if f(pm) {
			ret = append(ret, pm)
		}
	}
	return
}

func (p *Parser) selectFirstParam(f func(pm *param) bool) (*param, error) {
	pms := filterParams(p.params, f)
	if len(pms) == 0 {
		return nil, nil
	}
	return pms[0], nil
}

func (p *Parser) selectOneParam(f func(pm *param) bool) (*param, error) {
	pms := filterParams(p.params, f)
	switch len(pms) {
	case 0:
		return nil, nil
	case 1:
		return pms[0], nil
	default:
		return nil, fmt.Errorf("matched multiple params: %v", pms)
	}
}

func (p *Parser) ParseOne() (err error) {
	arg := (*p.args)[0]
	log.Printf("processing %q", arg)
	//if arg == "--" {
	//	return p.parsePositionalOnly(params...)
	//}
	*p.args = (*p.args)[1:]
	if len(arg) > 2 && arg[:2] == "--" {
		match, err := p.selectOneParam(func(pm *param) bool {
			if pm.positional {
				return false
			}
			for _, l := range pm.long {
				if l == arg[2:] {
					return true
				}
			}
			return false
		})
		if err != nil {
			return err
		}
		if match == nil {
			return fmt.Errorf("unmatched switch %q", arg)
		}
		if match.run != nil {
			p.RanSubCmd = true
			return match.run(SubCmdCtx{
				unusedArgs: p.args,
				parent:     p,
			})
		}
		*p.args, err = match.parse((*p.args)[:])
		return err
	}
	subcmd, err := p.selectOneParam(func(pm *param) bool {
		if !pm.positional {
			return false
		}
		for _, l := range pm.long {
			if l == arg {
				return true
			}
		}
		return false
	})
	if err != nil {
		return
	}
	if subcmd != nil {
		err = subcmd.run(SubCmdCtx{unusedArgs: p.args})
		if err != nil {
			err = fmt.Errorf("running subcommand %q: %w", subcmd.name, err)
		}
		p.RanSubCmd = true
		return
	}
	pos, err := p.selectFirstParam(func(pm *param) bool {
		if !pm.positional {
			return false
		}
		return pm.valid && len(pm.short) == 0 && len(pm.long) == 0
	})
	if err != nil {
		return err
	}
	if pos != nil {
		*p.args, err = pos.parse(append([]string{arg}, *p.args...))
		if err != nil {
			err = fmt.Errorf("parsing %v: %w", pos.name, err)
		}
		return
	}
	return fmt.Errorf("unexpected argument: %q, choices: %v", arg, p.params)
}

func (p *Parser) eachChoice(each func(c Param)) {
	for _, choice := range p.params {
		each(choice)
	}
}

func (p *Parser) PrintChoices(w io.Writer) {
	p.eachChoice(func(c Param) {
		u := c.Usage()
		fmt.Fprintf(w, "  ")
		if len(u.Switches) != 0 {
			fmt.Fprintf(w, "%v ", strings.Join(u.Switches, ","))
		}
		for _, arg := range u.Arguments {
			fmt.Fprintf(w, "<%v>", arg)
		}
		fmt.Fprintf(w, "\n")
		if u.Help != "" {
			fmt.Fprintf(w, "\t%v\n", u.Help)
		}
	})
}

type SubCmdCtx struct {
	unusedArgs *[]string
	parent     *Parser
	err        error
}

func NewParser() *Parser {
	return &Parser{
		params: []*param{HelpFlag()},
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

func Subcommand(name string, run SubcommandRunner) *param {
	return &param{
		run:        run,
		name:       name,
		long:       []string{name},
		positional: true,
	}
}

func FromStruct(target interface{}) (params []Param) {
	value := reflect.ValueOf(target).Elem()
	type_ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		target := fieldValue.Addr().Interface()
		structField := type_.Field(i)
		pm := &param{
			name:       structField.Name,
			target:     target,
			positional: structField.Tag.Get("arg") == "positional",
			valid:      true,
			help:       structField.Tag.Get("help"),
		}
		arity := structField.Tag.Get("arity")
		switch target.(type) {
		case *bool, **bool:
			pm.nullary = true
			pm.parse = func(args []string) (unusedArgs []string, err error) {
				return args, unmarshalInto("true", target)
			}
			pm.satisfied = true
		default:
			pm.parse = func(args []string) (unusedArgs []string, err error) {
				if arity == "+" {
					pm.satisfied = true
				}
				return args[1:], unmarshalInto(args[0], target)
			}
		}
		if !pm.positional {
			pm.long = []string{xstrings.ToKebabCase(structField.Name)}
		}
		switch arity {
		case "":
			pm.satisfied = true
		case "+":
		default:
			panic(fmt.Sprintf("unhandled arity %q on %v", arity, type_))
		}
		params = append(params, pm)
	}
	return
}

func ParseMain(params ...Param) *Parser {
	return Parse(os.Args[1:], params...)
}

func (p *Parser) AddParams(params ...Param) *Parser {
	for _, pm := range params {
		p.params = append(p.params, pm.(*param))
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
