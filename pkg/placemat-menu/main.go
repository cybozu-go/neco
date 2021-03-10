package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cybozu-go/neco/menu"
)

var (
	flagConfig = flag.String("f", "", "Template file for placemat-menu")
	flagOutDir = flag.String("o", ".", "Directory for output files")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	configFile, err := filepath.Abs(*flagConfig)
	if err != nil {
		return err
	}
	inputDir := filepath.Dir(configFile)

	outputDir, err := filepath.Abs(*flagOutDir)
	if err != nil {
		return err
	}

	f, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := menu.Parse(bufio.NewReader(f), inputDir)
	if err != nil {
		return err
	}

	cluster, err := menu.NewCluster(m)
	if err != nil {
		return err
	}

	if err := cluster.Generate(inputDir, outputDir); err != nil {
		return err
	}

	return nil
}
