package args

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type Parser struct {
	args      *[]string
	params    []*param
	RanSubCmd bool
	deferred  []func() error
	posOnly   bool
}

func (p *Parser) Parse() error {
	for len(*p.args) > 0 {
		arg := (*p.args)[0]
		err := p.ParseOne()
		if err != nil {
			return fmt.Errorf("parsing %q: %w", arg, err)
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

func (p *Parser) runParam(pm *param) (err error) {
	p.RanSubCmd = true
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		control, ok := r.(subCmdParseErr)
		if !ok {
			panic(r)
		}
		err = control.err
	}()
	err = pm.run(SubCmdCtx{
		unusedArgs: p.args,
		parent:     p,
		deferred:   &p.deferred,
	})
	return
}

func (p *Parser) ParseOne() (err error) {
	arg := (*p.args)[0]
	//log.Printf("processing %q", arg)
	//if arg == "--" {
	//	return p.parsePositionalOnly(params...)
	//}
	*p.args = (*p.args)[1:]
	if !p.posOnly {
		if arg == "--" {
			p.posOnly = true
			return nil
		}
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
				return p.runParam(match.param)
			}
			err = p.doParse(match.param, *p.args, match.negative)
			return err
		}
		if arg[0] == '-' {
			return errors.New("short flags not yet supported")
		}
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
		err = p.runParam(subcmd.param)
		if err != nil {
			err = fmt.Errorf("running subcommand %q: %w", subcmd.param.name, err)
		}
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
			fmt.Fprintf(w, "%v", strings.Join(u.Switches, "|"))
		}
		for _, arg := range u.Arguments {
			fmt.Fprintf(w, " <%v>", arg)
		}
		fmt.Fprintf(w, "\n")
		if u.Help != "" {
			fmt.Fprintf(w, "\t%v\n", u.Help)
		}
	})
}

func NewParser() *Parser {
	return &Parser{
		params: []*param{HelpFlag()},
	}
}

func (p *Parser) SetArgs(args *[]string) {
	p.args = args
}

func (p *Parser) AddParams(params ...Param) *Parser {
	for _, pm := range params {
		p.params = append(p.params, pm.(*param))
	}
	return p
}
