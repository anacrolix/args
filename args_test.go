package args

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func ExampleLongHelp() {
	flag := Flag(FlagOpt{
		Long:   "flag",
		Target: nil,
		Short:  0,
	})
	Parse([]string{"--help"}, flag).Run()
	// Output:
	// valid arguments at this point:
	//   --help|-h
	//   --flag
}

func TestLeadingHyphenArg(t *testing.T) {
	flag := Flag(FlagOpt{Long: "flag", Default: true})
	var arg string
	pos := Pos("arg", &arg)
	r := Parse([]string{"-no-flag", "actual"}, flag)
	c := qt.New(t)
	c.Check(r.Err, qt.IsNotNil)

	r = Parse([]string{"--", "-no-flag"}, flag, pos)
	c.Check(r.Err, qt.IsNil)
	c.Check(arg, qt.Equals, "-no-flag")
	c.Check(flag.Bool(), qt.IsTrue)

	r = Parse([]string{"--", "-no-flag", "actual"}, flag)
	c.Log(r.Err)
	c.Check(r.Err, qt.IsNotNil)
}

func TestStructPositional(t *testing.T) {
	c := qt.New(t)
	var s struct {
		One  string   `arg:"positional"`
		Plus []string `arg:"positional" arity:"*"`
	}
	c.Check(Parse(nil, FromStruct(&s)...).Err, qt.IsNotNil)

	c.Check(Parse([]string{"first", "second", "third"}, FromStruct(&s)...).Err, qt.IsNil)
	c.Check(s.One, qt.Equals, "first")
	c.Check(s.Plus, qt.DeepEquals, []string{"second", "third"})
}
