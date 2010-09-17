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
	output io.Writer
}

type input struct {
	beats uint
	beatlength uint
}
	

func makeeffecthandlerstruct(chan frame) C.struct_sox_effects_handler_t {
	
}

func create(effp *C.struct_sox_effect_t, argc int, argv **C.char) int {

}

func start(effp *C.struct_sox_effect_t) int {
}

func openband(filename string, band int) chan frame {
	// this function 
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
			info.beatlength = getfilesamples(filename) / info.beats
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
			var channel chan frame = openband(filename, band)
			f := <-channel
			trackdata[i / beatlength][band].left += fabs(float64(f.left))
			trackdata[i / beatlength][band].right += fabs(float64(f.right))
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
	flag.IntVar(&config.bands, 20, "bands", "number of bands")
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
}

func main() {
	config := new(config)
	config.readflags()
	config.writeoutput()
}
