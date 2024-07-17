package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"rsc.io/script"
)

var (
	printLog = flag.Bool("log", false, "Show output log")
	printOut = flag.Bool("stdout", false, "Print stdout of last command")
	printErr = flag.Bool("stderr", false, "Print stderr of last command")
)

func main() {
	var (
		help        = flag.Bool("help", false, "Show help")
		cmd         = flag.String("c", "", "Single command to run")
		interactive = flag.Bool("i", false, "Interactive mode")
	)

	flag.Parse()

	if *help {
		fmt.Fprintln(os.Stderr, "gscript [options] scripts...")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(0)
	}

	// process command argument
	if *cmd != "" {
		stdout, stderr, scrlog, err := runScript("command", strings.NewReader(*cmd))

		print(stdout, stderr, scrlog)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// process scripts
	for i, scr := range flag.Args() {
		stdout, stderr, scrlog, err := runFile(scr)

		print(stdout, stderr, scrlog)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(i + 1)
		}
	}

	// process stdin if there are not scripts and -c was not provided
	if flag.NArg() == 0 && *cmd == "" {
		if *interactive {
			scrlog, err := runInteractive("stdin", os.Stdin)

			if *printLog {
				fmt.Println(scrlog)
			}

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			stdout, stderr, scrlog, err := runScript("stdin", os.Stdin)

			print(stdout, stderr, scrlog)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

	}
}

func print(stdout, stderr, scrlog string) {
	if *printLog {
		fmt.Println(scrlog)
	} else {
		if *printOut && len(stdout) > 0 {
			if *printErr {
				fmt.Println("[stdout]")
			}
			fmt.Println(stdout)
		}
		if *printErr && len(stderr) > 0 {
			if *printOut {
				fmt.Fprintln(os.Stderr, "[stderr]")
			}
			fmt.Fprintln(os.Stderr, stderr)
		}
	}
}

func runFile(name string) (stdout string, stderr string, scrlog string, err error) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", "", "", err
	}
	defer f.Close()

	return runScript(name, f)
}

func runScript(name string, scriptRdr io.Reader) (stdout string, stderr string, scrlog string, err error) {
	engine := script.NewEngine()
	configEngine(engine)
	var state *script.State
	state, err = script.NewState(context.Background(), ".", os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	reader := bufio.NewReader(scriptRdr)
	var logf bytes.Buffer
	err = engine.Execute(state, name, reader, &logf)
	scrlog = logf.String()
	stdout = state.Stdout()
	stderr = state.Stderr()
	return
}

func runInteractive(name string, scriptRdr io.Reader) (string, error) {
	engine := script.NewEngine()
	configEngine(engine)
	state, err := script.NewState(context.Background(), ".", os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}
	var logf bytes.Buffer

	scanner := bufio.NewScanner(scriptRdr)
	for scanner.Scan() {
		if scanner.Text() == "stop" {
			break
		}
		reader := bufio.NewReader(strings.NewReader(scanner.Text()))
		err = engine.Execute(state, name, reader, &logf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			stdout := state.Stdout()
			stderr := state.Stderr()
			if len(stdout) > 0 {
				if len(stderr) > 0 {
					fmt.Println("[stdout]")
				}
				fmt.Println(stdout)
			}
			if len(stderr) > 0 {
				if len(stdout) > 0 {
					fmt.Fprintln(os.Stderr, "[stderr]")
				}
				fmt.Fprintln(os.Stderr, stderr)
			}
		}
	}

	return logf.String(), nil
}

func configEngine(engine *script.Engine) {
	engine.Cmds["execx"] = execExpand("execx", engine.Cmds["exec"])
	engine.Conds["file"] = fileExists()
	engine.Conds["env"] = envIsSet()
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
