package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"rsc.io/script"
)

func main() {
	var (
		printLog = flag.Bool("log", false, "Show output log")
		printOut = flag.Bool("stdout", false, "Print stdout of last command")
		printErr = flag.Bool("stderr", false, "Print stderr of last command")
	)

	flag.Parse()

	for _, scr := range flag.Args() {
		stdout, stderr, scrlog, err := runScript(scr)

		if *printLog {
			fmt.Println(scrlog)
		}
		if *printOut {
			fmt.Println(stdout)
		}
		if *printErr {
			fmt.Fprintln(os.Stderr, stderr)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runScript(scr string) (stdout string, stderr string, scrlog string, err error) {
	engine := script.NewEngine()
	// engine.ListCmds(os.Stdout, true)
	// engine.ListConds(os.Stdout, nil)
	engine.Cmds["execx"] = execExpand("execx", engine.Cmds["exec"])
	engine.Conds["file"] = fileExists()
	engine.Conds["env"] = envIsSet()
	var state *script.State
	state, err = script.NewState(context.Background(), ".", os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	f, err := os.Open(scr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var logf bytes.Buffer
	err = engine.Execute(state, scr, reader, &logf)
	scrlog = logf.String()
	stdout = state.Stdout()
	stderr = state.Stderr()
	return
}

func execExpand(name string, execCmd script.Cmd) script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: "run an executable program with arguments",
			Args:    "program [args...]",
			Detail: []string{
				"Note that '" + name + "' does not terminate the script (unlike Unix shells).",
				"Unlike 'exec', arguments with spaces will be expanded into separate arguments.",
			},
			Async: true,
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) < 1 {
				return nil, script.ErrUsage
			}
			var newArgs []string
			for _, arg := range args {
				list := strings.Split(strings.ReplaceAll(arg, "\t", " "), " ")
				for _, s := range list {
					s = strings.TrimSpace(s)
					if s != "" {
						newArgs = append(newArgs, s)
					}
				}
			}
			return execCmd.Run(s, newArgs...)
		})
}

func fileExists() script.Cond {
	return script.PrefixCondition(
		"<suffix> is a file that exists",
		func(_ *script.State, suffix string) (bool, error) {
			fi, err := os.Stat(suffix)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
			return fi != nil, err
		})
}

func envIsSet() script.Cond {
	return script.PrefixCondition(
		"<suffix> is an environment variable that is set and non-blank",
		func(s *script.State, suffix string) (bool, error) {
			e, _ := s.LookupEnv(suffix)
			return strings.TrimSpace(e) != "", nil
		})
}
