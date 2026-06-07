package main

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
