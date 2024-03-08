package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"rsc.io/script"
)

func main() {
	var (
		printLog = flag.Bool("log", false, "Show output log")
		printOut = flag.Bool("stdout", false, "Print stdout")
		printErr = flag.Bool("stderr", false, "Print stderr")
	)

	flag.Parse()

	for _, scr := range flag.Args() {
		engine := script.NewEngine()
		// engine.ListCmds(os.Stdout, true)
		// engine.ListConds(os.Stdout, nil)
		state, err := script.NewState(context.Background(), ".", os.Environ())
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
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			// fmt.Fprintln(os.Stderr, logf.String())
			return
		}
		if *printLog {
			fmt.Println(logf.String())
		}
		if *printOut {
			fmt.Println(state.Stdout())
		}
		if *printErr {
			fmt.Fprintln(os.Stderr, state.Stderr())
		}
	}
}
