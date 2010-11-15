package main

//#include "sox.h"

import (
	"flag"
	"fmt"
	"os"
	"exec"
	"bufio"
	"remixoscope"
)

// assume we want stereo, 16-bit output. sample rate is adjustable
// we get back 32-bit sox samples

var filetoclose *os.File = nil

func readflags(config *remixoscope.Config) {
	flag.StringVar(&config.Inputlist, "inputlist", "inputlist", "list of input files with metadata")
	flag.UintVar(&config.Bands, "bands", 10, "number of bands")
	soxpath, _ := exec.LookPath("sox")
	flag.StringVar(&config.Sox, "sox", soxpath, "Path to sox binary. Default is /usr/bin/sox")
	o := flag.String("output", "-", "output file. Use \"-\" for stdout.")
	flag.Parse()
	if *o == "-" {
		config.Output = bufio.NewWriter(os.Stdout)
	} else {
		f, err := os.Open(*o, os.O_CREAT|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			panic(fmt.Sprintf("couldn't open file %s for output! error %s", *o, err.String()))
		}
		fmt.Fprintf(os.Stderr, "File open, descriptor is %d\n", f.Fd())
		config.Output = bufio.NewWriter(f)
		filetoclose = f
	}

	config.Soxopts = make([]string, 0)
	if flag.NArg() > 0 {
		config.Soxopts = append(config.Soxopts, flag.Args()...)
	} else {
		config.Soxopts = append(config.Soxopts, []string{"-b", "16", "-e", "signed-integer", "-B", "-r", "44100", "-t", "raw"}...)
	}
}

func main() {
	config := new(remixoscope.Config)
	readflags(config)
	config.Writeanalysis()
	if filetoclose != nil {
		fmt.Fprintln(os.Stderr, "closing file")
		filetoclose.Close()
	}
}
