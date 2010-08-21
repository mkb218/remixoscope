package main

import (
	"flag"
	"io"
	"os"
	"hash"
	md5 "crypto/md5"
)

type samples [][][][]uint16

type stats struct {
	beatlength int
	grid       [][][]uint
}

type trackinfo_t struct {
	filename string
	length   int
	md5sum   hash.Hash
	stats    stats
}

func processtracklist(filename string) []trackinfo_t {
	f, err := file.Open(filename)
	if f == nil {
		panic(fmt.Sprintf("can't open %s: error %s\n", filename, err))
	}
	var key string
	var value string
	while n, err := fmt.Fscanf(f, "%s %s\n"), 
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		panic("not enough args")
	}

	tracklist := flag.Arg(0)
	inputdir := flag.Arg(1)

	trackinfo := processtracklist(tracklist)
}
