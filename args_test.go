package args

import (
	"testing"

	"github.com/go-quicktest/qt"
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
	qt.Check(t, qt.IsNotNil(r.Err))

	r = Parse([]string{"--", "-no-flag"}, flag, pos)
	qt.Check(t, qt.IsNil(r.Err))
	qt.Check(t, qt.Equals(arg, "-no-flag"))
	qt.Check(t, qt.IsTrue(flag.Bool()))

	r = Parse([]string{"--", "-no-flag", "actual"}, flag)
	t.Log(r.Err)
	qt.Check(t, qt.IsNotNil(r.Err))
}

func TestStructPositional(t *testing.T) {
	var s struct {
		One  string   `arg:"positional"`
		Plus []string `arg:"positional" arity:"*"`
	}
	qt.Check(t, qt.IsNotNil(Parse(nil, FromStruct(&s)...).Err))

	qt.Check(t, qt.IsNil(Parse([]string{"first", "second", "third"}, FromStruct(&s)...).Err))
	qt.Check(t, qt.Equals(s.One, "first"))
	qt.Check(t, qt.DeepEquals(s.Plus, []string{"second", "third"}))
}
