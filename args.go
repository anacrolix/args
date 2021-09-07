package args

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"encoding"
)

type Arg interface{
	Match(s string) (string, bool)
	Parse(value string, next []string) ([]string, error)
}

type subcommand struct{
	Name string
	run func([]string) ([]string, error)
}

func (cmd subcommand) Match(s string) (string, bool) {
	return "", s==cmd.Name
}

func (cmd subcommand) Parse(value string, next []string) ([]string, error) {
	return cmd.run(next)
}

func Subcommand(name string, run func([]string)([]string,error)) Arg {
	return subcommand{Name: name, run: run}
}

type FlagOpt struct{
	target *bool
	Long string
	Short rune
}

func NewFlag(opt FlagOpt) Flag {
	return Flag{opt: opt}
}

type Flag struct{
	opt FlagOpt
}

func (f Flag) Value() bool {
	return f.opt.target != nil && *f.opt.target
}

type pos[T any] struct{
	target *T
}

type PosTarget interface{
	~string | encoding.TextUnmarshaler
}

type Option struct {
	Name string
	Target interface{}
}

type Pos struct{
	Target *PosTarget
}


type parser struct{
	args []string
}

func NewMainParser() *parser {
	return &parser{os.Args[1:]}
}

func (p *parser) ParseAll(args ...Arg) error {
	for len(p.args) > 0 {
		err := p.ParseOne(args...)
		if err != nil {
			return err
		}
	}
	return nil
}

type Match struct{
	Input string
	Arg Arg
	Value string
}

func (p *parser) ParseOne(args...Arg) error {
	matches := make([]Match, 0, 1)
	for _, arg := range args {
		value, ok :=  arg.Match(p.args[0])
		if ok {
			matches = append(matches, Match{p.args[0], arg,value})
		}
	}
	switch len(matches) {
	case 0:
		return errors.New("unmatched argument")
	default:
		return fmt.Errorf("matched multiple parameters: %v", matches)
	case 1:
	}
	match := matches[0]
	p.args=p.args[1:]
	left, err := match.Arg.Parse(match.Value, p.args)
	if err != nil {
		return fmt.Errorf("parsing %v: %w", match.Arg, err)
	}
	p.args=left
	return nil
}

func (p *parser) MustParse(args ...Arg) {
	err := p.ParseAll(args...)
	if err != nil {
		panic(err)
	}
}

func FromStruct(target interface{}) (args []Arg) {
	value := reflect.ValueOf(target)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := value.Type().Field(i)
		args = append(args, Option{
			Name: structField.Name,
			Target: field.Addr().Interface(),
		})
	}
	return
}
