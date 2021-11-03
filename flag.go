package args

type FlagOpt struct {
	Long   string
	Target *bool
	Short  rune
	// Only used if Target is nil
	Default bool
}

// Flags are switches that don't take arguments.
func Flag(opts FlagOpt) *param {
	if opts.Target == nil {
		opts.Target = new(bool)
		*opts.Target = opts.Default
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
