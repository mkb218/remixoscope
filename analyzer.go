package main

//#include "sox.h"

import (
	"strings"
	"strconv"
	"flag"
	"fmt"
	"os"
	"json"
	"exec"
	"bytes"
	"regexp"
	"math"
	"bufio"
	vector "container/vector"
	ioutil "io/ioutil"
)

// assume we want stereo, 16-bit output. sample rate is adjustable
// we get back 32-bit sox samples

type soxsample int32
type frame struct {
	left, right float64
}

type beat struct {
	frames []frame // one frame per band
}
type track struct {
	beats []beat
}

type config struct {
	inputlist string
	bands uint
	sox string
	soxopts vector.StringVector
	output *bufio.Writer
}

type input struct {
	beats uint
	beatlength uint
	channels uint
}

type FileWriter string

var filetoclose *os.File = nil

func AppendSlice(this * vector.StringVector, append []string) {
	for _, s := range append {
		this.Push(s)
	}
}

func openband(config *config, remixspec *vector.StringVector, filename string, band uint) (datachan chan frame, quitchan chan bool) {
	bandwidth := 22050 / config.bands
	bandlow := band * bandwidth
	bandhigh := bandlow + bandwidth
	
	currsoxopts := make(vector.StringVector, 0)
	currsoxopts.Push("sox")
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	currsoxopts.Push(filename)
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	currsoxopts.AppendVector(&config.soxopts)
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	currsoxopts.Push("-")
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	currsoxopts.AppendVector(remixspec)
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	currsoxopts.Push("sinc")
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	
	if bandhigh >= 22050 {
		currsoxopts.Push(strconv.Uitoa(bandlow))
		fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	} else {
		currsoxopts.Push(strconv.Uitoa(bandlow)+"-"+strconv.Uitoa(bandhigh))
		fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	}
	
	AppendSlice(&currsoxopts, []string{"channels", "2"})
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	
	getwd, _ := os.Getwd()
	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	p, err := exec.Run(config.sox, currsoxopts, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.PassThrough)
	if err != nil {
		panic(fmt.Sprintf("couldn't open band %d for reason %s", band, err))
	}
	fmt.Fprintf(os.Stderr, "sox pid is %d\n", p.Pid)
	// some day this will use libsox
	datachan = make(chan frame)
	quitchan = make(chan bool)
	go func() {	
		buf := make([]byte, 4)
		for {
			size, err := p.Stdout.Read(buf)
			if err != nil {
				if size == 0 && err == os.EOF {
					break
				} else {
					panic(fmt.Sprintf("error reading from sox! %s", err))
				}
			}
		
			frame := new(frame)
			frame.left = float64(buf[0])*256+float64(buf[1])
			frame.right = float64(buf[2])*256+float64(buf[1])
			datachan <- *frame
		}
		fmt.Fprintln(os.Stderr, "Done reading file")
		quitchan <- true
		fmt.Fprintln(os.Stderr, "Quit message sent. Sox out.")
	} ()
	return datachan, quitchan
}

func (config *config) getfileinfo(filename string) (samplelength uint, channels uint) {
	getwd, _ := os.Getwd()
	p, err := exec.Run(config.sox, []string{"soxi", filename}, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.Pipe)
	if err != nil {
		panic(fmt.Sprintf("couldn't open soxi on file %s! %s", filename, err))
	}
	
	var soxierr []byte
	soxierr, err = ioutil.ReadAll(p.Stderr)
	if err != nil {
		panic(fmt.Sprintf("Error reading soxi stderr %s", err))
	}
	if len(soxierr) > 0 {
		panic(fmt.Sprintf("soxi had stderr %s", soxierr))
	}
	var soxiout []byte
	soxiout, err = ioutil.ReadAll(p.Stdout)
	if err != nil {
		panic(fmt.Sprintf("Error reading soxi stdout %s", err))
	}
	
	err = p.Close()
	if err != nil {
		panic(fmt.Sprintf("soxi returned err %s", err))
	}

	durexp := regexp.MustCompile("^Duration.* ([0-9]+) samples")
	chanexp := regexp.MustCompile("^Channels.* ([0-9]+)")
	for _, line := range strings.Split(string(soxiout), "\n", -1) {
		var err os.Error = nil
		if (durexp.MatchString(line)) {
			samplelength, err = strconv.Atoui(durexp.FindStringSubmatch(line)[1])
		} else if (chanexp.MatchString(line)) {
			channels, err = strconv.Atoui(chanexp.FindStringSubmatch(line)[1])
		}
		if err != nil {
			panic(fmt.Sprintf("bad int returned from soxi! %s: %s", line, err))
		}
	}
	
	return samplelength, channels
}

// we need one goroutine per each band to read samples, and one goroutine to read from each channel

// for each band
// channels[i] = openband(file, i, width)

func (config *config) readinputlist() map[string]input {
	in := false
	tracks := make(map[string]input)
	var filename string
	var info input
	b, err := ioutil.ReadFile(config.inputlist)
	filestuff := bytes.NewBuffer(b).String()
	if err != nil {
		panic(fmt.Sprintf("error reading inputlist %s\n", err))
	}
	lines := strings.Split(filestuff, "\n", -1)
	for _, line := range lines {
		if strings.HasPrefix(line, "BEGIN ") { 
			in = true
			filename = (strings.Split(line, " ", 2))[1]
		} else if line == "END" {
			in = false
			tracks[filename] = info
		} else if strings.HasPrefix(line, "LENGTH ") {
			lengthstr := (strings.Split(line, " ", 2))[1]
			info.beats, err = strconv.Atoui(lengthstr)
			if err != nil {
				panic(fmt.Sprintf("bad int in inputlist length %s: %s", lengthstr, err))
			}
			var samplelength uint
			samplelength, info.channels = config.getfileinfo(filename)
			info.beatlength = samplelength / info.beats
		}
	}
	if in { 
		panic(fmt.Sprintf("unfinished business reading input list", err))
	}
	return tracks
}

func (config *config) writeoutput() {
	t := config.readinputlist()
	tracks := make(map[string]track)
	for filename, info := range t {
		fmt.Fprintf(os.Stderr, "starting file %s\n", filename)
		remixspec := make(vector.StringVector, 0)
		if info.channels == 1 {
			AppendSlice(&remixspec, []string{"remix", "1", "1"})
			// stereo is a noop
			// everything >2 channels doesn't have enough information so I am assuming the layout based on mpeg standards
		} else if info.channels == 3 {
			AppendSlice(&remixspec, []string{"remix", "1,3", "2,3"})
		} else if info.channels == 4 {
			AppendSlice(&remixspec, []string{"remix", "1,3,4", "2,3,4"})
		} else if info.channels == 5 {
			AppendSlice(&remixspec, []string{"remix", "1,3,4", "2,3,5"})
		} else if info.channels == 6 { // 5.1
			AppendSlice(&remixspec, []string{"remix", "1,3,4,5", "2,3,4,6"})
		} else if info.channels == 7 { // 6.1
			AppendSlice(&remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,7"})
		} else if info.channels == 8 { // 7.1
			AppendSlice(&remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,8"})
		} else if info.channels > 8 { // no idea, just take first two
			AppendSlice(&remixspec, []string{"remix", "1", "2"})
		}

//		tracks = tracks[:1+len(tracks)]
		// loop over bands
		var trackdata track
		trackdata.beats = make([]beat,info.beats)
		for index := uint(0); index < info.beats; index++ {
			trackdata.beats[index].frames = make([]frame, config.bands)
		}
		for band := uint(0); band < config.bands; band++ {
			var i uint = 0
			datachan, quitchan := openband(config, &remixspec, filename, band)
			fmt.Fprint(os.Stderr, "got channels\n")
			L: for {
				select {
				case f := <-datachan:
					if i % 1000 == 0 {
						fmt.Fprintf(os.Stderr, "%d\n", i)
					}
					trackdata.beats[i / info.beatlength].frames[band].left += math.Fabs(float64(f.left))
					trackdata.beats[i / info.beatlength].frames[band].right += math.Fabs(float64(f.right))
				case b := <-quitchan:
					fmt.Fprintf(os.Stderr, "got quitchan msg %t\n", b)
					break L;
				}
				i++
			}
		}
		tracks[filename] = trackdata
	}
	outbytes, err := json.Marshal(tracks)
	if err != nil {
		panic(fmt.Sprintf("couldn't marshal JSON for tracks! SCREAM AND SHOUT %s", err.String()))
	}
	
	fmt.Fprintf(os.Stderr, "%d\n", len(outbytes))
	var written int
	written, err = config.output.Write(outbytes)
	
	if err != nil {
		panic(fmt.Sprintf("error writing bytes. written %d err %s\n", written, err))
	} else {
		fmt.Fprintf(os.Stderr, "written %d\n", written)
	}
	err = config.output.Flush()
	if err != nil {
		panic(fmt.Sprintf("flushing output failed! %s",err))
	}
}

func (config *config) readflags() {
	flag.StringVar(&config.inputlist, "inputlist", "inputlist", "list of input files with metadata")
	flag.UintVar(&config.bands, "bands", 10, "number of bands")
	flag.StringVar(&config.sox, "sox", "/usr/bin/sox", "Path to sox binary. Default is /usr/bin/sox")
	o := flag.String("output", "-", "output file. Use \"-\" for stdout.")
	flag.Parse()
	if *o == "-" {
		config.output = bufio.NewWriter(os.Stdout)
	} else {
		f, err := os.Open(*o, os.O_CREAT|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			panic(fmt.Sprintf("couldn't open file %s for output! error %s", *o, err.String()))
		}
		fmt.Fprintf(os.Stderr, "File open, descriptor is %d\n", f.Fd())
		config.output = bufio.NewWriter(f)
		filetoclose = f
	}
	
	config.soxopts = make(vector.StringVector, 0)
	if flag.NArg() > 0 {
		AppendSlice(&config.soxopts, flag.Args())
	} else {
		AppendSlice(&config.soxopts, []string{"-b","16","-e","signed-integer","-B", "-r","44100","-t","raw"})
	}
}

func main() {
	config := new(config)
	config.readflags()
	config.writeoutput()
	if filetoclose != nil {
		fmt.Fprintln(os.Stderr, "closing file")
		filetoclose.Close()
	}
}
