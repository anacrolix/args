package args

// Flags are switches that don't take arguments.
func Flag(opts FlagOpt) *param {
	if opts.Target == nil {
		opts.Target = new(bool)
	}
	pm := &param{
		target:    opts.Target,
		long:      []string{opts.Long},
		nullary:   true,
		satisfied: true,
		parse:     boolFlagParser(opts.Target),
		negative:  "no",
		valid:     true,
	}
	if opts.Short != 0 {
		pm.short = append(pm.short, opts.Short)
	}
	return pm
}

func boolFlagParser(target interface{}) paramParser {
	return func(args []string, negative bool) (unusedArgs []string, err error) {
		s := "true"
		if negative {
			s = "false"
		}
		return args, unmarshalInto(s, target)
	}
}
