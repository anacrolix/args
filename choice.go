package args

import (
	"fmt"
	"strings"
)

type Choice struct {
	Long     string
	Choices  map[string]interface{}
	Default  string
	selected string
}

func (c *Choice) ToParam() *param {
	c.selected = c.Default
	pm := &param{
		target:    &c.selected,
		long:      []string{c.Long},
		name:      c.Long,
		help:      strings.Join(c.ChoicesKeys(), "|"),
		satisfied: c.Default != "",
	}
	pm.parse = func(args []string, negative bool) (unusedArgs []string, err error) {
		if len(args) < 1 {
			err = fmt.Errorf("missing value: %v", pm.help)
			return
		}
		if _, ok := c.Choices[args[0]]; !ok {
			err = fmt.Errorf("invalid choice %q: %v", args[0], pm.help)
			return
		}
		c.selected = args[0]
		pm.satisfied = true
		return args[1:], nil
	}
	return pm
}

func (c *Choice) ChoicesKeys() []string {
	keys := make([]string, 0, len(c.Choices))
	for k := range c.Choices {
		keys = append(keys, k)
	}
	return keys
}

func (c *Choice) SelectedValue() interface{} {
	return c.Choices[c.selected]
}
