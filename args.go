package args

import (
	"encoding"
	"fmt"
	"io"
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

func AfterParse(f func() error) ParamOpt {
	return func(p *param) {
		p.afterParse = append(p.afterParse, f)
	}
}

type ParamOpt func(*param)

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

func filterParams(pms []*param, f func(pm *param) match) (ret []match) {
	for _, pm := range pms {
		m := f(pm)
		if m.ok {
			m.param = pm
			ret = append(ret, m)
		}
	}
	return
}

func (p *Parser) selectFirstParam(f func(pm *param) match) (match, error) {
	pms := filterParams(p.params, f)
	if len(pms) == 0 {
		return match{ok: false}, nil
	}
	return pms[0], nil
}

func (p *Parser) selectOneParam(f func(pm *param) match) (match, error) {
	pms := filterParams(p.params, f)
	switch len(pms) {
	case 0:
		return match{ok: false}, nil
	case 1:
		return pms[0], nil
	default:
		return match{ok: false}, fmt.Errorf("matched multiple params: %v", pms)
	}
}

type match struct {
	*param
	negative bool
	ok       bool
}

func (p *Parser) doParse(pm *param, args []string, negative bool) (err error) {
	*p.args, err = pm.parse(args, negative)
	if err != nil {
		return
	}
	for _, ap := range pm.afterParse {
		err = ap()
		if err != nil {
			err = fmt.Errorf("running after parse hook: %w", err)
		}
	}
	return
}

func (p *Parser) ParseOne() (err error) {
	arg := (*p.args)[0]
	//log.Printf("processing %q", arg)
	//if arg == "--" {
	//	return p.parsePositionalOnly(params...)
	//}
	*p.args = (*p.args)[1:]
	if len(arg) > 2 && arg[:2] == "--" {
		match, err := p.selectOneParam(func(pm *param) match {
			if pm.positional {
				return match{ok: false}
			}
			for _, l := range pm.long {
				if l == arg[2:] {
					return match{ok: true}
				}
				if pm.negative != "" {
					if pm.negative+"-"+l == arg[2:] {
						return match{negative: true, ok: true}
					}
				}
			}
			return match{ok: false}
		})
		if err != nil {
			return err
		}
		if !match.ok {
			return fmt.Errorf("unmatched switch %q", arg)
		}
		if match.param.run != nil {
			p.RanSubCmd = true
			return match.param.run(SubCmdCtx{
				unusedArgs: p.args,
				parent:     p,
			})
		}
		err = p.doParse(match.param, *p.args, match.negative)
		return err
	}
	pos, err := p.selectFirstParam(func(pm *param) match {
		if !pm.positional {
			return match{ok: false}
		}
		return match{ok: pm.valid && len(pm.short) == 0 && len(pm.long) == 0}
	})
	if err != nil {
		return err
	}
	if pos.ok {
		err = p.doParse(pos.param, append([]string{arg}, *p.args...), pos.negative)
		if err != nil {
			err = fmt.Errorf("parsing %v: %w", pos.name, err)
		}
		return
	}
	subcmd, err := p.selectOneParam(func(pm *param) match {
		if !pm.positional {
			return match{ok: false}
		}
		for _, l := range pm.long {
			if l == arg {
				return match{ok: true}
			}
		}
		return match{ok: false}
	})
	if err != nil {
		return
	}
	if subcmd.ok {
		err = subcmd.param.run(SubCmdCtx{unusedArgs: p.args})
		if err != nil {
			err = fmt.Errorf("running subcommand %q: %w", subcmd.param.name, err)
		}
		p.RanSubCmd = true
		return
	}
	return errUnexpectedArg{
		params: p.params,
		arg:    arg,
	}
}

func (p *Parser) eachChoice(each func(c *param)) {
	for _, choice := range p.params {
		each(choice)
	}
}

func (p *Parser) PrintChoices(w io.Writer) {
	fmt.Fprintf(w, "valid arguments at this point:\n")

	p.eachChoice(func(pm *param) {
		if !pm.valid {
			return
		}
		u := pm.Usage()
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
		satisfied:  true,
		nullary:    true,
		valid:      true,
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
			pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
				s := "true"
				if negative {
					s = "false"
				}
				return args, unmarshalInto(s, target)
			}
			pm.satisfied = true
			pm.negative = "no"
		default:
			pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
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
	p := NewParser()
	p.SetArgs(&args)
	p.AddParams(params...)
	p.Err = p.Parse()
	return p
}

func FatalUsage() {
	os.Exit(2)
}
