package args

func ExampleLongHelp() {
	flag := Flag(FlagOpt{
		Long:   "flag",
		Target: nil,
		Short:  0,
	})
	Parse([]string{"--help"}, flag).Run()
	// Output:
	// valid arguments at this point:
	//   --help,-h
	//   --flag
}
