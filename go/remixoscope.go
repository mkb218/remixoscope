package main

import (
	"bufio"
	"fmt"
	"os"
	"math"
	binary "encoding/binary"
	"exec"
	"regexp"
	"strings"
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

func shuffle(v Vector) {
	for i := len(v) - 1; i >= 1; i-- {
		j := rand.Intn(i)
		sv.Swap(i, j)
	}
}

func marshal() (outstring []string) {
	// <bandcount>|trackname|beat0band0lbeat0band0r...Beat0bandNr
	// numbers aren't intended to be human readable, but it is easier to emit human readable integers
	out := make([]string, 0)
	out = append(out, fmt.Sprintf("%d", bands))
	for _, track := range sources {
		out = append(out, fmt.Sprintf("|%s|%d|", track.basename, len(track.beats)))
		if track.beats <= 128 {
			for i := uint(0); i < bands; i += 1 {
				for j := uint(0); j < len(track.beats); j += 1 {
					fmt.Fprintf(os.Stderr, "%d %d %f %f\n", i, j, track.beats[j].buckets[i].Left, track.beats[j].buckets[i].Right)
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
		panic(fmt.Sprintf("couldn't open soxi on file %s! %s", filename, err))
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
	for sampledone, channelsdone := false, false; !sampledone && !channelsdone; {
		line, err := soxiout.ReadString('\n')
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

	var filename string

	f, err := os.Open(inputlist, os.O_RDONLY, 0)
	if err != nil {
		panic(fmt.Sprintf("error reading inputlist %s\n", err))
	}

	b := bufio.NewReader(f)

	for {
		line := b.ReadString('\n')
		if strings.HasPrefix(line, "BEGIN ") {
			in = true
			filename = (strings.Split(line, " ", 2))[1]
		} else if line == "END" {
			in = false
			sources = append(sources, source{filename})
		} else if strings.HasPrefix(line, "LENGTH ") {
			lengthstr := (strings.Split(line, " ", 2))[1]
			info.Beats, err = strconv.Atoui(lengthstr)
			if err != nil {
				panic(fmt.Sprintf("bad int in inputlist length %s: %s", lengthstr, err))
			}
			var samplelength uint
			samplelength, info.Channels = Getfileinfo(config.Sox, filename)
			info.Beatlength = samplelength / info.Beats
		}
	}
	if in {
		panic(fmt.Sprintf("unfinished business reading input list", err))
	}
}

/*
func (config *Config) Writeanalysis() {
	t := config.readinputlist()
	tracks := make(map[string]Track)
	for filename, info := range t {
		fmt.Fprintf(os.Stderr, "starting file %s\n", filename)


		//		tracks = tracks[:1+len(tracks)]
		// loop over bands
		var trackdata Track
		trackdata.Beats = make([]Beat, info.Beats)
		trackdata.Info = &info
		for index := uint(0); index < info.Beats; index++ {
			trackdata.Beats[index].Buckets = make([]Bucket, config.Bands)
		}
		for band := uint(0); band < config.Bands; band++ {
			var i uint = 0
			datachan, quitchan := Openband(info.Channels, config.Sox, config.Soxopts, filename, band, config.Bands)
			fmt.Fprint(os.Stderr, "got channels\n")
			fmt.Fprintf(os.Stderr, "beatlength %d, band %d / %d, beats %d\n", info.Beatlength, band, config.Bands, info.Beats)
		L:
			for {
				select {
				case f := <-datachan:
					b := Bucket{float64(f.Left), float64(f.Right)}
					dex := i / info.Beatlength
					if i%10000 == 0 {
						fmt.Fprintf(os.Stderr, "%d %f %f\n", i, trackdata.Beats[dex].Buckets[band].Left, trackdata.Beats[dex].Buckets[band].Right)
					}
					if dex >= uint(len(trackdata.Beats)) {
						dex = uint(len(trackdata.Beats)) - 1
						// rolloff
						b.Left = b.Left * float64(dex / uint(len(trackdata.Beats)))
						b.Right = b.Right * float64(dex / uint(len(trackdata.Beats)))
					}
					trackdata.Beats[dex].Buckets[band].Left += math.Fabs(b.Left)
					trackdata.Beats[dex].Buckets[band].Right += math.Fabs(b.Right)
				case b := <-quitchan:
					fmt.Fprintf(os.Stderr, "got quitchan msg %t\n", b)
					break L
				}
				i++
			}
		}
/*		for beatno, _ := range trackdata.Beats {
			for bandno, _ := range trackdata.Beats[beatno].Buckets {
				trackdata.Beats[beatno].Buckets[bandno].Left = math.Sqrt(trackdata.Beats[beatno].Buckets[bandno].Left / float64(info.Beatlength))
				trackdata.Beats[beatno].Buckets[bandno].Right = math.Sqrt(trackdata.Beats[beatno].Buckets[bandno].Right / float64(info.Beatlength))
			}
		}

		tracks[filename] = trackdata
	}

	outbytes := config.marshal(tracks)

	fmt.Fprintf(os.Stderr, "%d\n", len(outbytes))

	for _, elem := range outbytes {
		written, err := config.Output.WriteString(elem)

		if err != nil {
			panic(fmt.Sprintf("error writing bytes. written %d err %s\n", written, err))
		}
	}
	err := config.Output.Flush()
	if err != nil {
		panic(fmt.Sprintf("flushing output failed! %s", err))
	}
}*/

func opensrcband(filename string, band, bands uint) chan buffer {
	bandwidth := 22050 / bands
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

	if bandhigh >= 22050 {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow))
	} else {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow)+"-"+strconv.Uitoa(bandhigh))
	}

	currsoxopts = append(currsoxopts, []string{"channels", "2"}...)

	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	cmd := startsox(sox, currsoxopts)
	// some day this will use libsox
	datachan = make(chan buffer, 5)
	go func() {
		for {
			var frame buffer
			err := binary.Read(cmd.Stdout, binary.BigEndian, &frame.left)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					fmt.Fprintln(os.Stderr, "Done reading file, closing datachan")
					close(datachan)
					break
				}
			}
			err = binary.Read(cmd.Stdout, binary.BigEndian, &frame.right)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					fmt.Fprintln(os.Stderr, "Done reading file, closing datachan")
					close(datachan)
					break
				}
			}
			datachan <- frame
			if closed(datachan) {
				fmt.Fprintln(os.Stderr, "Output channel closed, returning")
				break
			}
		}
	}()
	return datachan
}

func startsox(sox string, currsoxopts []string, outpipe bool) exec.Cmd {
	getwd, _ := os.Getwd()
	outstat := exec.Pipe
	if (!outpipe) {
		outstat = exec.DevNull
	}
	p, err := exec.Run(sox, currsoxopts, os.Environ(), getwd, exec.DevNull, outstat, exec.PassThrough)
	if err != nil {
		panic(fmt.Sprintf("couldn't open band for reason %s", err))
	}
	fmt.Fprintf(os.Stderr, "sox pid is %d\n", p.Pid)
	return p
}

func main() {
	readflags()
}
