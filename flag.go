package args

func Flag(opts FlagOpt) *param {
	pm := &param{
		target:  opts.Target,
		long:    []string{opts.Long},
		nullary: true,
	}
	if opts.Short != 0 {
		pm.short = append(pm.short, opts.Short)
	}
	return pm
}
