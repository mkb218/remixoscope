package main

//#include "sox.h"

import (
	"C"
	"strings"
	"strconv"
	"flag"
	"io"
	"os"
	"file"
	"math"
	"json"
	"exec"
	vector "container/vector"
	ioutil "io/ioutil"
)

// assume we want stereo, 16-bit output. sample rate is adjustable
// we get back 32-bit sox samples

type soxsample int32
type frame struct {
	left, right float64
}

type beat []frame // one frame per band
type track []beat

type config struct {
	inputlist string
	bands int
	sox string
	soxopts StringVector
	output io.Writer
}

type input struct {
	beats uint
	beatlength uint
	channels uint
}

func openband(config *config, remixspec StringVector, filename string, bandwidth, band int) chan frame {
	currsoxopts := make(StringVector, 1)
	currsoxopts.Push("sox")
	currsoxopts.Push(filename)
	currsoxopts.AppendVector(config.soxopts)
	currsoxopts.Push("-")
	currsoxopts.AppendVector(remixspec)
	p, err := exec.Run(config.sox, currsoxopts, os.Environ(), os.Getwd(), exec.DevNull, exec.Pipe, exec.DevNull)
	if err != nil {
		panic(fmt.Sprintf("couldn't open band %d for reason %s", band, err))
	}
	// some day this will use libsox
	outch := make(chan frame)
	go func (out chan frame) {	
		
	} (outch)
	return outch
}

func (config *config) getfileinfo(filename string) (samplelength uint, channels uint) {
	p, err := exec.Run(config.sox, ["soxi", filename], os.Environ(), os.Getwd(), exec.DevNull, exec.Pipe, exec.Pipe)
	if err != nil {
		panic(fmt.Sprintf("couldn't open soxi on file %s! %s", filename, err)
	}
	
	var soxierr, err = ioutil.ReadAll(p.Stderr)
	if err != nil {
		panic(fmt.Sprintf("Error reading soxi stderr %s", err))
	}
	var soxiout, err = ioutil.ReadAll(p.Stdout)
	if err != nil {
		panic(fmt.Sprintf("Error reading soxi stdout %s", err))
	}
	
	err = p.Close()
	if err != nil {
		panic(fmt.Sprintf("soxi returned err %s", err))
	}
	
	for line := range strings.Split(soxiout, 
}

// we need one goroutine per each band to read samples, and one goroutine to read from each channel

// for each band
// channels[i] = openband(file, i, width)

func (config *config) readinputlist() map[string]input {
	fr, err := file.Open(config.inputlist, O_RDONLY, 0)
	if err != 0 {
		panic(fmt.sprintf("Can't open input list file %s for writing! %s", config.inputlist, err.String()))
	}
	r := bufio.NewReader(fr)
	in := false
	tracks := map[string]input
	var filename string
	var info input
	for i := 0; err != 0 ;line, err := r.ReadString('\n') {
		if line.HasPrefix("BEGIN ") { 
			in = true
			_, filename = string.Split(line, " ", 2)
		} else if line == "END" {
			in = false
			tracks[filename] = info
		} else if line.HasPrefix("LENGTH ") {
			_, lengthstr := string.Split(line, " ", 2)
			info.beats = strconv.Atoui64(lengthstr)
			var samplelength, info.channels = config.getfileinfo(filename) / info.beats
			info.beatlength = samplelength / info.beats
		}
	}
	if in { 
		panic("unfinished business reading input list")
	}
	return tracks
}

func (config *config) writeoutput() {
	t := config.readinputlist()
	tracks := make([]track, len(t))
	for filename, info := range t {
		tracks = tracks[:1+len(tracks)]
		// loop over bands
		trackdata := make(track,info.beats)
		for beat := range trackdata {
			beat = make(beat, config.bands)
		}
		i := 0
		for band := 0; band < config.bands; band++ {
			var channel chan frame = openband(config, filename, band)
			var f
			for ; f != nil; f := <-channel {
				trackdata[i / beatlength][band].left += fabs(float64(f.left))
				trackdata[i / beatlength][band].right += fabs(float64(f.right))
			}
		}
		tracks[len(tracks)-1] = trackdata
	}
	outbytes, err := json.Marshal(tracks)
	if err != 0 {
		panic(fmt.sprintf("couldn't marshal JSON for tracks! SCREAM AND SHOUT %s", err.String()))
	}
	
	config.output.Write(outbytes)
}

func (config *config) readflags() {
	flag.StringVar(&config.inputlist, "inputlist", "inputlist", "list of input files with metadata")
	flag.IntVar(&config.bands, "bands", 20, "number of bands")
	flag.StringVar(&config.sox, "sox", "/usr/bin/sox", "Path to sox binary. Default is /usr/bin/sox")
	o := flag.String("output", "-", "output file. Use \"-\" for stdout.")
	flag.Parse()
	if *o == "-" {
		config.output = bufio.NewWriter(os.Stdout)
	} else {
		f, err := os.Open(*o, 0, 0)
		if err != 0 {
			panic(fmt.sprintf("couldn't open file %s for output! error %s", *s, err.String()))
		}
		config.output = bufio.NewWriter(f)
	}
	
	config.soxopts = make(vector.StringVector, 10)
	if flag.NArg() > 0 {
		config.soxopts.AppendVector(flag.Args())
	} else {
		config.soxopts.Push("-b")
		config.soxopts.Push("16")
		config.soxopts.Push("-e")
		config.soxopts.Push("signed-integer")
		config.soxopts.Push("-r")
		config.soxopts.Push("44100")
		config.soxopts.Push("-t")
		config.soxopts.Push("raw")
	}
}

func main() {
	config := new(config)
	config.readflags()
	config.writeoutput()
}
