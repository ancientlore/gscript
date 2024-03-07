package main

import (
	"bufio"
	"context"
	"flag"
	"os"

	"rsc.io/script"
)

func main() {
	var file = flag.String("script", "", "Script to run")
	flag.Parse()

	engine := script.NewEngine()
	// engine.ListCmds(os.Stdout, true)
	// engine.ListConds(os.Stdout, nil)
	state, err := script.NewState(context.Background(), ".", os.Environ())
	if err != nil {
		panic(err)
	}
	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	//var logf bytes.Buffer
	err = engine.Execute(state, *file, reader, os.Stdout)
	if err != nil {
		panic(err)
	}
	//fmt.Println(state.Stdout())
}
