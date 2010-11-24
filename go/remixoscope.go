package remixoscope

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
);
type Soxsample int32
type Frame struct {
	Left, Right int16
}

type Bucket struct {
	Left, Right float64
}

type Beat struct {
	Buckets []Bucket // one frame per band
}
type Track struct {
	Beats []Beat
	Info *Input
}

type Config struct {
	Inputlist string
	Bands     uint
	Sox       string
	Soxopts   []string
	Output    *bufio.Writer
}

type Input struct {
	Beats      uint
	Beatlength uint
	Channels   uint
}

type StringWriter struct {
	out *string
}

func (this *StringWriter) Write(p []byte) (n int, err os.Error) {
	tmp := fmt.Sprintf("%s", p)
	this.out = &tmp
	return len(tmp), nil
}

func (config *Config) marshal(tracks map[string]Track) (outstring []string) {
	// <bandcount>|trackname|beat0band0lbeat0band0r...Beat0bandNr
	// numbers aren't intended to be human readable, but it is easier to emit human readable integers
	out := make([]string, 0)
	out = append(out, fmt.Sprintf("%d", config.Bands))
	for trackname, track := range tracks {
		out = append(out, fmt.Sprintf("|%s|%d|", trackname, track.Info.Beats))
		if track.Info.Beats <= 128 {
			for i := uint(0); i < config.Bands; i += 1 {
				for j := uint(0); j < track.Info.Beats; j += 1 {
					fmt.Fprintf(os.Stderr, "%d %d %f %f\n", i, j, track.Beats[j].Buckets[i].Left, track.Beats[j].Buckets[i].Right)
				}
			}
		}
		for _, beat := range track.Beats {
			for _, band := range beat.Buckets {
				var l uint64 = math.Float64bits(band.Left)
				var r uint64 = math.Float64bits(band.Right)
				var sw StringWriter
				binary.Write(&sw, binary.BigEndian, l)
				out = append(out, *sw.out)
				binary.Write(&sw, binary.BigEndian, r)
				out = append(out, *sw.out)
			}
		}
	}
	return out
}

func getfileinfo(sox, filename string) (samplelength uint, channels uint) {
	getwd, _ := os.Getwd()
	p, err := exec.Run(sox, []string{"soxi", filename}, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.Pipe)
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
		if durexp.MatchString(line) {
			samplelength, err = strconv.Atoui(durexp.FindStringSubmatch(line)[1])
		} else if chanexp.MatchString(line) {
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

func (config *Config) readinputlist() map[string]Input {
	in := false
	tracks := make(map[string]Input)
	var filename string
	var info Input
	b, err := ioutil.ReadFile(config.Inputlist)
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
			info.Beats, err = strconv.Atoui(lengthstr)
			if err != nil {
				panic(fmt.Sprintf("bad int in inputlist length %s: %s", lengthstr, err))
			}
			var samplelength uint
			samplelength, info.Channels = config.getfileinfo(filename)
			info.Beatlength = samplelength / info.Beats
		}
	}
	if in {
		panic(fmt.Sprintf("unfinished business reading input list", err))
	}
	return tracks
}

func (config *Config) Writeanalysis() {
	t := config.readinputlist()
	tracks := make(map[string]Track)
	for filename, info := range t {
		fmt.Fprintf(os.Stderr, "starting file %s\n", filename)
		remixspec := make([]string, 0)
		if info.Channels == 1 {
			remixspec = append(remixspec, []string{"remix", "1", "1"}...)
			// stereo is a noop
			// everything >2 channels doesn't have enough information so I am assuming the layout based on mpeg standards
		} else if info.Channels == 3 {
			remixspec = append(remixspec, []string{"remix", "1,3", "2,3"}...)
		} else if info.Channels == 4 {
			remixspec = append(remixspec, []string{"remix", "1,3,4", "2,3,4"}...)
		} else if info.Channels == 5 {
			remixspec = append(remixspec, []string{"remix", "1,3,4", "2,3,5"}...)
		} else if info.Channels == 6 { // 5.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5", "2,3,4,6"}...)
		} else if info.Channels == 7 { // 6.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,7"}...)
		} else if info.Channels == 8 { // 7.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,8"}...)
		} else if info.Channels > 8 { // no idea, just take first two
			remixspec = append(remixspec, []string{"remix", "1", "2"}...)
		}

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
			datachan, quitchan := config.openband(remixspec, filename, band)
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
		}*/
		
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
}

func (config *Config) openband(remixspec []string, filename string, band uint) (datachan chan Frame, quitchan chan bool) {
	bandwidth := 22050 / config.Bands
	bandlow := band * bandwidth
	bandhigh := bandlow + bandwidth

	currsoxopts := make([]string, 0)
	currsoxopts = append(currsoxopts, "sox")
	currsoxopts = append(currsoxopts, filename)
	currsoxopts = append(currsoxopts, config.Soxopts...)
	currsoxopts = append(currsoxopts, "-")
	currsoxopts = append(currsoxopts, remixspec...)
	currsoxopts = append(currsoxopts, "sinc")

	if bandhigh >= 22050 {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow))
	} else {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow) + "-" + strconv.Uitoa(bandhigh))
	}

	currsoxopts = append(currsoxopts, []string{"channels", "2"}...)

	fmt.Fprintln(os.Stderr, strings.Join(currsoxopts, " "))
	return Startsox(config.Sox, currsoxopts)
}

func Startsox(sox string, currsoxopts []string) (datachan chan Frame, quitchan chan bool) {
	getwd, _ := os.Getwd()
	p, err := exec.Run(sox, currsoxopts, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.PassThrough)
	if err != nil {
		panic(fmt.Sprintf("couldn't open band for reason %s", err))
	}
	fmt.Fprintf(os.Stderr, "sox pid is %d\n", p.Pid)
	// some day this will use libsox
	datachan = make(chan Frame)
	quitchan = make(chan bool)
	go func() {
		for {
			frame := new(Frame)
			err := binary.Read(p.Stdout, binary.BigEndian, &frame.Left)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					break
				}
			}
			err = binary.Read(p.Stdout, binary.BigEndian, &frame.Right)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					break 
				}
			}
			datachan <- *frame
		}
		fmt.Fprintln(os.Stderr, "Done reading file")
		quitchan <- true
		fmt.Fprintln(os.Stderr, "Quit message sent. Sox out.")
	}()
	return datachan, quitchan
}
