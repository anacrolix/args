package args

type ParamOpt func(*param)

func Help(helpText string) ParamOpt {
	return func(p *param) {
		p.help = helpText
	}
}

func AfterParse(f func() error) ParamOpt {
	return func(p *param) {
		p.afterParse = append(p.afterParse, f)
	}
}

func Arity(arity byte) ParamOpt {
	return func(p *param) {
		p.arity = arity
	}
}
