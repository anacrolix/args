package args

type OptOpt struct {
	Long     string
	Target   interface{}
	Short    rune
	Required bool
	Arity    string
}

// Options are switches that take arguments.
func Opt(opts OptOpt) *param {
	if opts.Target == nil {
		panic("opt target must not be nil")
	}
	pm := &param{
		target:    opts.Target,
		long:      []string{opts.Long},
		nullary:   false,
		satisfied: !opts.Required,
		valid:     true,
	}
	if opts.Short != 0 {
		pm.short = append(pm.short, opts.Short)
	}
	pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
		if opts.Arity == "+" {
			pm.satisfied = true
		}
		return args[1:], unmarshalInto(args[0], opts.Target)
	}
	return pm
}
