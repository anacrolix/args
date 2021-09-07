package args

import (
	"fmt"
	"strings"
)

func Flag(opts FlagOpt) *flag {
	return &flag{
		target: opts.Target,
		long:   opts.Long,
		short:  opts.Short,
	}
}

type flag struct {
	target *bool
	long   string
	short  rune
	run    func(*Parser) error
}

func (f *flag) Usage() string {
	var sb strings.Builder
	if f.short != 0 {
		fmt.Fprintf(&sb, "-%c", f.short)
	}
	if f.long != "" {
		if sb.Len() != 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "--%v", f.long)
	}
	return sb.String()
}

func (f *flag) Long() []string {
	return []string{f.long}
}

func (f *flag) Parse(args []string) ([]string, error) {
	if f.target == nil {
		f.target = new(bool)
	}
	*f.target = true
	return args, nil
}

func (f *flag) Value() bool {
	return f.target != nil && *f.target
}
