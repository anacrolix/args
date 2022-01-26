package args

type SubCmdCtx struct {
	unusedArgs *[]string
	parent     *Parser
	deferred   *[]func() error
}

func (me *SubCmdCtx) Defer(f func() error) {
	*me.deferred = append(*me.deferred, f)
}

func (me *SubCmdCtx) NewParser() *Parser {
	p := NewParser()
	p.SetArgs(me.unusedArgs)
	return p
}

// Parses given params and aborts and returns the subcommand context on an error. Returns *Parser in
// case caller wants to examine conditions of the parse. Errors are otherwise handled automatically.
func (me *SubCmdCtx) Parse(params ...Param) *Parser {
	p := me.NewParser()
	p.AddParams(params...)
	err := p.Parse()
	if err != nil {
		panic(subCmdParseErr{err})
	}
	return p
}

type subCmdParseErr struct {
	err error
}

type SubcommandRunner func(sub SubCmdCtx) (err error)

func Subcommand(name string, run SubcommandRunner, opts ...ParamOpt) *param {
	pm := &param{
		run:        run,
		name:       name,
		long:       []string{name},
		positional: true,
		satisfied:  true,
		nullary:    true,
		valid:      true,
	}
	for _, opt := range opts {
		opt(pm)
	}
	return pm
}
