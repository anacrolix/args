package args

import (
	"errors"

	"github.com/huandu/xstrings"
)

type OptOpt struct {
	Long     string
	Target   interface{}
	Short    rune
	Required bool
}

// Options are switches that take arguments.
func Opt(opts OptOpt) *param {
	if opts.Target == nil {
		panic("opt target must not be nil")
	}
	pm := &param{
		target:    opts.Target,
		long:      []string{xstrings.ToKebabCase(opts.Long)},
		nullary:   false,
		satisfied: !opts.Required,
		valid:     true,
	}
	if opts.Short != 0 {
		pm.short = append(pm.short, opts.Short)
	}
	pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
		pm.satisfied = true
		if len(args) < 1 {
			return args, errors.New("insufficient arguments")
		}
		return args[1:], unmarshalInto(args[0], opts.Target)
	}
	return pm
}
