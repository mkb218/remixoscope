package main

import (
	"bufio"
	"path"
	"flag"
	"rand"
	"fmt"
	"syscall"
	"os"
	"math"
	binary "encoding/binary"
	"exec"
	"regexp"
	"strings"
	vector "container/vector"
	"strconv"
	"bytes"
	ioutil "io/ioutil"
)

type buffer struct {
	left, right []int16
}

type bucket struct {
	left, right float64
}

type beat struct {
	buckets []bucket // one frame per band
}

type source struct {
	filename string
	beats    []beat
}

var sources []source

var soxpath string
var soxformatopts []string
var tmpdir string
var outputdir string
var beatlength uint
var bands uint
var outputext string
var inputfiles []string
var samplerate uint
var buffersize uint

func shuffle(v vector.Vector) {
	for i := len(v) - 1; i >= 1; i-- {
		j := rand.Intn(i)
		v.Swap(i, j)
	}
}

func marshal() (outstring []string) {
	// <bandcount>|trackname|beat0band0lbeat0band0r...Beat0bandNr
	// numbers aren't intended to be human readable, but it is easier to emit human readable integers
	out := make([]string, 0)
	out = append(out, fmt.Sprintf("%d", bands))
	for _, track := range sources {
		out = append(out, fmt.Sprintf("|%s|%d|", track.filename, len(track.beats)))
		if len(track.beats) <= 128 {
			for i := uint(0); i < bands; i += 1 {
				for j := uint(0); j < uint(len(track.beats)); j += 1 {
					//					fmt.Fprintf(os.Stderr, "%d %d %f %f\n", i, j, track.beats[j].buckets[i].left, track.beats[j].buckets[i].right)
				}
			}
		}
		for _, beat := range track.beats {
			for _, band := range beat.buckets {
				l := math.Float64bits(band.left)
				r := math.Float64bits(band.right)
				var sb bytes.Buffer
				binary.Write(&sb, binary.BigEndian, l)
				binary.Write(&sb, binary.BigEndian, r)
				out = append(out, sb.String())
			}
		}
	}
	return out
}

func getfileinfo(filename string) (samplelength uint, channels uint) {
	getwd, _ := os.Getwd()
	p, err := exec.Run(soxpath, []string{"soxi", filename}, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.Pipe)
	if err != nil {
		panic(fmt.Sprintf("couldn't open soxi %s on file %s! %s", soxpath, filename, err))
	}

	soxierr, err := ioutil.ReadAll(p.Stderr)
	if err != nil {
		panic(fmt.Sprintf("Error reading soxi stderr %s", err))
	}
	if len(soxierr) > 0 {
		panic(fmt.Sprintf("soxi had stderr %s", soxierr))
	}

	soxiout := bufio.NewReader(p.Stdout)

	durexp := regexp.MustCompile("^Duration.* ([0-9]+) samples")
	chanexp := regexp.MustCompile("^Channels.* ([0-9]+)")
	for sampledone, channelsdone := false, false; !sampledone || !channelsdone; {
		line, err := soxiout.ReadString('\n')
		line = strings.TrimSpace(line)
		fmt.Println(line)
		if durexp.MatchString(line) {
			samplelength, err = strconv.Atoui(durexp.FindStringSubmatch(line)[1])
			sampledone = true
		} else if chanexp.MatchString(line) {
			channels, err = strconv.Atoui(chanexp.FindStringSubmatch(line)[1])
			channelsdone = true
		}
		if err != nil {
			panic(fmt.Sprintf("bad int returned from soxi! %s: %s", line, err))
		}
	}

	err = p.Close()
	if err != nil {
		panic(fmt.Sprintf("soxi returned err %s", err))
	}

	return samplelength, channels
}


// we need one goroutine per each band to read samples, and one goroutine to read from each channel

// for each band
// channels[i] = openband(file, i, width)

func readinputlist(inputlist string) {
	in := false

	f, err := os.Open(inputlist, os.O_RDONLY, 0)
	if err != nil {
		panic(fmt.Sprintf("error reading inputlist %s\n", err))
	}

	b := bufio.NewReader(f)

	var s *source
	done := false
	for !done {
		line, err := b.ReadString('\n')
		if err != nil {
			if err == os.EOF {
				done = true
			} else {
				panic(err)
			}
		}
		line = strings.TrimSpace(line)
		fmt.Fprintln(os.Stderr, line)
		if strings.HasPrefix(line, "BEGIN ") {
			in = true
			s = new(source)
			s.filename = (strings.Split(line, " ", -1))[1]
		} else if line == "END" {
			in = false
			sources = append(sources, *s)
		} else if strings.HasPrefix(line, "LENGTH ") {
			lengthstr := (strings.Split(line, " ", -1))[1]
			length, err := strconv.Atoui(lengthstr)
			s.beats = make([]beat, length)
			if err != nil {
				panic(fmt.Sprintf("bad int in inputlist length %s: %s", lengthstr, err))
			}
		}
	}
	if in {
		panic(fmt.Sprintf("unfinished business reading input list", err))
	}
}

func (b buffer) empty() bool {
	return !((b.left != nil) || (b.right != nil))
}

func analyze(s *source) {
	fmt.Fprintf(os.Stderr, "starting file %s\n", s.filename)
	//		tracks = tracks[:1+len(tracks)]
	// loop over bands
	for index := 0; index < len(s.beats); index++ {
		s.beats[index].buckets = make([]bucket, bands)
	}

	for band := uint(0); band < bands; band++ {
		i := uint(0)
		var bi uint
		samplelength, datachan := opensrcband(s.filename, band)
		beatlength := samplelength / uint(len(s.beats))
		fmt.Fprint(os.Stderr, "got channels\n")
		fmt.Fprintf(os.Stderr, "beatlength %d, band %d / %d, beats %d\n", beatlength, band, bands, len(s.beats))
		for {
			f := <-datachan
			if f.empty() && closed(datachan) {
				break
			}
			bi = 0
			//			fmt.Println(samplelength, len(s.beats))
			dex := i / beatlength
			defer func() {
				e := recover()
				if e != nil {
					fmt.Println(bi, beatlength, dex, len(s.beats), len(f.left), len(f.right))
					panic(e)
				}
			}()
			for ; (dex < uint(len(s.beats)) && bi < beatlength) && bi < uint(len(f.left)); bi++ {
				b := bucket{float64(f.left[bi]), float64(f.right[bi])}
				if dex >= uint(len(s.beats)) {
					dex = uint(len(s.beats)) - 1
					// rolloff
					b.left = b.left * float64(dex/uint(len(s.beats)))
					b.right = b.right * float64(dex/uint(len(s.beats)))
				}
				s.beats[dex].buckets[band].left += math.Fabs(b.left)
				s.beats[dex].buckets[band].right += math.Fabs(b.right)
				//				if i%10000 == 0 {
				fmt.Fprintf(os.Stderr, "%d %d %f %d %f %f %f\n", i, f.left[bi], float64(f.left[bi]), f.right[bi], float64(f.right[bi]), s.beats[dex].buckets[band].left, s.beats[dex].buckets[band].right)
				//				}

				i++
			}
		}
	}
}

func opensrcband(filename string, band uint) (uint, <-chan buffer) {
	bandwidth := samplerate / 2 / bands
	bandlow := band * bandwidth
	bandhigh := bandlow + bandwidth

	samplelength, channels := getfileinfo(filename)

	currsoxopts := make([]string, 0)
	currsoxopts = append(currsoxopts, "sox")
	currsoxopts = append(currsoxopts, filename)
	currsoxopts = append(currsoxopts, soxformatopts...)
	currsoxopts = append(currsoxopts, "-")

	if channels == 1 {
		currsoxopts = append(currsoxopts, []string{"remix", "1", "1"}...)
		// stereo is a noop
		// everything >2 channels doesn't have enough information so I am assuming the layout based on mpeg standards
	} else if channels == 3 {
		currsoxopts = append(currsoxopts, []string{"remix", "1,3", "2,3"}...)
	} else if channels == 4 {
		currsoxopts = append(currsoxopts, []string{"remix", "1,3,4", "2,3,4"}...)
	} else if channels == 5 {
		currsoxopts = append(currsoxopts, []string{"remix", "1,3,4", "2,3,5"}...)
	} else if channels == 6 { // 5.1
		currsoxopts = append(currsoxopts, []string{"remix", "1,3,4,5", "2,3,4,6"}...)
	} else if channels == 7 { // 6.1
		currsoxopts = append(currsoxopts, []string{"remix", "1,3,4,5,7", "2,3,4,6,7"}...)
	} else if channels == 8 { // 7.1
		currsoxopts = append(currsoxopts, []string{"remix", "1,3,4,5,7", "2,3,4,6,8"}...)
	} else if channels > 8 { // no idea, just take first two
		currsoxopts = append(currsoxopts, []string{"remix", "1", "2"}...)
	}

	currsoxopts = append(currsoxopts, "sinc")

	fmt.Println(bandhigh, samplerate/2/bands)
	if bandhigh >= samplerate/2/bands {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow))
	} else {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow)+"-"+strconv.Uitoa(bandhigh))
	}

	currsoxopts = append(currsoxopts, []string{"channels", "2"}...)

	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	cmd := startsox(soxpath, currsoxopts, true)
	// some day this will use libsox
	datachan := make(chan buffer, 5)
	go func() {
	OUTER:
		for {
			var frame buffer
			frame.left = make([]int16, buffersize)
			frame.right = make([]int16, buffersize)
			for i := uint(0); i < buffersize; i++ {
				err := binary.Read(cmd.Stdout, binary.BigEndian, &(frame.left[i]))
				if err != nil {
					if err != os.EOF {
						panic(err)
					} else {
						fmt.Fprintf(os.Stderr, "Done reading file at L %d, closing datachan\n", i)
						frame.left = frame.left[0:i]
						frame.right = frame.right[0:i]
						datachan <- frame
						close(datachan)
						break OUTER
					}
				}
				err = binary.Read(cmd.Stdout, binary.BigEndian, &(frame.right[i]))
				if err != nil {
					if err != os.EOF {
						panic(err)
					} else {
						fmt.Fprintf(os.Stderr, "Done reading file R %d, closing datachan\n", i)
						frame.left = frame.left[0:i]
						frame.right = frame.right[0:i]
						datachan <- frame
						close(datachan)
						break OUTER
					}
				}
				fmt.Fprintln(os.Stderr, i, frame.left[i], frame.right[i])
			}
			datachan <- frame
			if closed(datachan) {
				fmt.Fprintln(os.Stderr, "Output channel closed, returning")
				break
			}
		}
		cmd.Close()
		close(datachan)
	}()
	return samplelength, datachan
}

func startsox(sox string, currsoxopts []string, outpipe bool) *exec.Cmd {
	getwd, _ := os.Getwd()
	outstat := exec.Pipe
	if !outpipe {
		outstat = exec.PassThrough
	}
	fmt.Println(currsoxopts)
	p, err := exec.Run(sox, currsoxopts, os.Environ(), getwd, exec.DevNull, outstat, exec.PassThrough)
	if err != nil {
		panic(fmt.Sprintf("couldn't open band for reason %s", err))
	}
	fmt.Fprintf(os.Stderr, "sox pid is %d\n", p.Pid)
	return p
}

func readflags() *string {
	sourcelist := flag.String("sourcelist", "sourcelist.txt", "list of source files with metadata")
	flag.UintVar(&bands, "bands", 10, "number of bands")
	var err os.Error
	soxpath, err = exec.LookPath("sox")
	defaults := "Default is " + soxpath
	checksoxpath := false
	if err != nil {
		checksoxpath = true
		defaults = "No sox found in path. No default"
	}

	flag.StringVar(&soxpath, "sox", soxpath, "Path to sox binary. "+defaults)
	flag.UintVar(&samplerate, "samplerate", 44100, "Sample rate in hz. Default 44100")
	flag.UintVar(&beatlength, "beatlength", 0, "Length of output beats in samples. No default")
	flag.StringVar(&tmpdir, "tmpdir", "/tmp", "Tmpdir to hold FIFOs. Must exist.")
	flag.StringVar(&outputdir, "outputdir", ".", "Dir to hold output files. Default is working directory")
	flag.StringVar(&outputext, "outputext", "remix.wav", "Output default extension. Do NOT start this with a dot. Default is remix.wav")

	flag.Parse()

	if checksoxpath && soxpath == "" {
		fmt.Fprintln(os.Stderr, "No sox found on PATH and no sox specified")
	}

	if beatlength == 0 {
		fmt.Fprintln(os.Stderr, "No beatlength specified.")
		os.Exit(1)
	}

	if samplerate == 0 && !(samplerate == 22050 || samplerate == 44100 || samplerate == 48000) {
		fmt.Fprintln(os.Stderr, "Bad samplerate specified")
		os.Exit(1)
	}

	stat, err := os.Stat(tmpdir)
	if err != nil || !stat.IsDirectory() {
		fmt.Fprintf(os.Stderr, "tmpdir %s does not exist or is not a directory: %s.\n", tmpdir, err.String())
		os.Exit(1)
	}

	stat, err = os.Stat(outputdir)
	if err != nil || !stat.IsDirectory() {
		fmt.Fprintf(os.Stderr, "outputdir %s does not exist or is not a directory: %s.\n", outputdir, err.String())
		os.Exit(1)
	}

	soxformatopts = append(soxformatopts, []string{"-b", "16", "-e", "signed-integer", "-B", "-r", strconv.Uitoa(samplerate), "-t", "raw"}...)

	buffersize = 512
	return sourcelist
}

func main() {
	sourcelist := readflags()
	readinputlist(*sourcelist)
	// phase 1, analyze all sources
	for i, _ := range sources {
		analyze(&sources[i])
		fmt.Fprintln(os.Stderr, "done analyze")
		generate(&sources[i])
	}
}

func generate(s *source) {
	fmt.Println("generate")
	outfilename := s.filename + "." + outputext
	basename := path.Base(outfilename)
	outfilename = path.Join(outputdir, basename)
	fifos := make([]string, 0, bands)
	for i := uint(0); i < bands; i++ {
		// make a fifo
		fifoname := path.Join(tmpdir, basename+strconv.Uitoa(i))
		fmt.Fprintf(os.Stderr, "making fifo %d\n", i)
		err := os.RemoveAll(fifoname)
		if err != nil {
			panic(err)
		}
		errno := syscall.Mkfifo(fifoname, 0600)
		if errno != 0 {
			panic(os.NewSyscallError("Mkfifo", errno))
		}
		fifos = append(fifos, fifoname)
		// make a goroutine to read from the channel with processinputband and write to the fifo
		// open the fifo inside the goroutine so the call to open doesn't block the main thread
		go processinputband(s, fifoname, i, openinputband(i))
	}

	// start sox with all the fifos
	// bands is number of files to read from
	// +1 output file
	// +1 for "-m"
	// +1 for "sox" at start
	opts := make([]string, 0, uint(len(soxformatopts))+bands+3)
	opts = append(opts, "sox", "-m")
	for _, f := range fifos {
		opts = append(opts, soxformatopts...)
		opts = append(opts, f)
	}
	opts = append(opts, outfilename)
	cmd := startsox(soxpath, opts, false)
	msg, err := cmd.Wait(0)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg.String())
}

func openinputband(band uint) <-chan buffer {
	sfile := flag.Arg(rand.Intn(flag.NArg() - 1))
	outchan := make(chan buffer, 5)
	go func() {
		for {
			_, inchan := opensrcband(sfile, band)
			for {
				buffer := <-inchan
				if closed(inchan) || closed(outchan) {
					break
				}
				outchan <- buffer
			}
			if closed(outchan) {
				close(inchan)
				break
			}
		}
	}()
	return outchan
}

func processinputband(s *source, fifoname string, band uint, channel <-chan buffer) {
	fifo, err := os.Open(fifoname, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}

	defer func() {
		eek := recover()
		err := fifo.Close()
		if err != nil {
			panic(err)
		}
		if eek != nil {
			panic(eek)
		}
	}()

	origbeats := s.beats
	for _, b := range s.beats {
		fmt.Println(b)
	}
	buckets := make([]bucket, len(s.beats))
	buffers := make([]buffer, len(s.beats))
	for i := 0; i < len(buffers); i++ {
		buffers[i].left = make([]int16, beatlength)
		buffers[i].right = make([]int16, beatlength)
	}
	inbuf := <-channel
	inbufpos := 0
	beatpos := uint(0)
	done := false
	for !done {
		beat := beatpos / beatlength
		buffpos := beatpos % beatlength
		if buckets[beat].left < origbeats[beat].buckets[band].left && buckets[beat].right < origbeats[beat].buckets[band].right {
			buffers[beat].left[buffpos] += inbuf.left[inbufpos]
			buckets[beat].left += float64(inbuf.left[inbufpos])
			buffers[beat].right[buffpos] += inbuf.right[inbufpos]
			buckets[beat].right += float64(inbuf.right[inbufpos])
			inbufpos++
		}
		if uint(inbufpos) >= beatlength {
			inbufpos = 0
			inbuf = <-channel
		}
		beatpos++
		if beatpos >= beatlength {
			beatpos = 0
		}
		done = true
		for _, b := range buckets {
			if !(b.left < origbeats[beat].buckets[band].left && b.right < origbeats[beat].buckets[band].right) {
				done = false
				break
			}
		}
	}
	for _, b := range buffers {
		binary.Write(fifo, binary.BigEndian, b.left)
		binary.Write(fifo, binary.BigEndian, b.right)
	}
}
