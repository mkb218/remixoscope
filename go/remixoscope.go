package remixoscope

import (
	"bufio"
	"fmt"
	"os"
);
type Soxsample int32
type Frame struct {
	Left, Right float64
}

type Beat struct {
	Frames []frame // one frame per band
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
	Sutput    *bufio.Writer
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

func (config *config) marshal(tracks map[string]track) (outstring []string) {
	// <bandcount>|trackname|beat0band0lbeat0band0r...beat0bandNr
	// numbers aren't intended to be human readable, but it is easier to emit human readable integers
	out := make([]string, 0)
	out = append(out, fmt.Sprintf("%d", config.bands))
	for trackname, track := range tracks {
		out = append(out, fmt.Sprintf("|%s|%d|", trackname, track.info.beats))
		if track.info.beats <= 128 {
			for i := uint(0); i < config.bands; i += 1 {
				for j := uint(0); j < track.info.beats; j += 1 {
					fmt.Fprintf(os.Stderr, "%d %d %f %f\n", i, j, track.beats[j].frames[i].left, track.beats[j].frames[i].right)
				}
			}
		}
		for _, beat := range track.beats {
			for _, band := range beat.frames {
				var l uint64 = math.Float64bits(band.left)
				var r uint64 = math.Float64bits(band.right)
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

func (config *config) Writeanalysis() {
	t := config.readinputlist()
	tracks := make(map[string]track)
	for filename, info := range t {
		fmt.Fprintf(os.Stderr, "starting file %s\n", filename)
		remixspec := make([]string, 0)
		if info.channels == 1 {
			remixspec = append(remixspec, []string{"remix", "1", "1"})
			// stereo is a noop
			// everything >2 channels doesn't have enough information so I am assuming the layout based on mpeg standards
		} else if info.channels == 3 {
			remixspec = append(remixspec, []string{"remix", "1,3", "2,3"})
		} else if info.channels == 4 {
			remixspec = append(remixspec, []string{"remix", "1,3,4", "2,3,4"})
		} else if info.channels == 5 {
			remixspec = append(remixspec, []string{"remix", "1,3,4", "2,3,5"})
		} else if info.channels == 6 { // 5.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5", "2,3,4,6"})
		} else if info.channels == 7 { // 6.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,7"})
		} else if info.channels == 8 { // 7.1
			remixspec = append(remixspec, []string{"remix", "1,3,4,5,7", "2,3,4,6,8"})
		} else if info.channels > 8 { // no idea, just take first two
			remixspec = append(remixspec, []string{"remix", "1", "2"})
		}

		//		tracks = tracks[:1+len(tracks)]
		// loop over bands
		var trackdata track
		trackdata.beats = make([]beat, info.beats)
		trackdata.info = &info
		for index := uint(0); index < info.beats; index++ {
			trackdata.beats[index].frames = make([]frame, config.bands)
		}
		for band := uint(0); band < config.bands; band++ {
			var i uint = 0
			datachan, quitchan := openband(config, &remixspec, filename, band)
			fmt.Fprint(os.Stderr, "got channels\n")
			fmt.Fprintf(os.Stderr, "beatlength %d, band %d / %d, beats %d\n", info.beatlength, band, config.bands, info.beats)
		L:
			for {
				select {
				case f := <-datachan:
					dex := i / info.beatlength
					if i%10000 == 0 {
						fmt.Fprintf(os.Stderr, "%d %f %f\n", i, trackdata.beats[dex].frames[band].left, trackdata.beats[dex].frames[band].right)
					}
					if dex >= uint(len(trackdata.beats)) {
						dex = uint(len(trackdata.beats)) - 1
						// rolloff
						f.left = f.left * float64(dex / uint(len(trackdata.beats)))
						f.right = f.right * float64(dex / uint(len(trackdata.beats)))
					}
					trackdata.beats[dex].frames[band].left += math.Fabs(f.left)
					trackdata.beats[dex].frames[band].right += math.Fabs(float64(f.right))
				case b := <-quitchan:
					fmt.Fprintf(os.Stderr, "got quitchan msg %t\n", b)
					break L
				}
				i++
			}
		}
/*		for beatno, _ := range trackdata.beats {
			for bandno, _ := range trackdata.beats[beatno].frames {
				trackdata.beats[beatno].frames[bandno].left = math.Sqrt(trackdata.beats[beatno].frames[bandno].left / float64(info.beatlength))
				trackdata.beats[beatno].frames[bandno].right = math.Sqrt(trackdata.beats[beatno].frames[bandno].right / float64(info.beatlength))
			}
		}*/
		
		tracks[filename] = trackdata
	}

	outbytes := config.marshal(tracks)

	fmt.Fprintf(os.Stderr, "%d\n", len(outbytes))

	outbytes.Do(func(elem string) {
		written, err := config.output.WriteString(elem)

		if err != nil {
			panic(fmt.Sprintf("error writing bytes. written %d err %s\n", written, err))
		}
	})
	err := config.output.Flush()
	if err != nil {
		panic(fmt.Sprintf("flushing output failed! %s", err))
	}
}

func (config *Config) openband(remixspec []string, filename string, band uint) (datachan chan frame, quitchan chan bool) {
	bandwidth := 22050 / config.bands
	bandlow := band * bandwidth
	bandhigh := bandlow + bandwidth

	currsoxopts := make([]string, 0)
	currsoxopts = append(currsoxopts, "sox")
	currsoxopts = append(currsoxopts, filename)
	currsoxopts = append(currsoxopts, config.soxopts)
	currsoxopts = append(currsoxopts, "-")
	currsoxopts = append(currsoxopts, remixspec)
	currsoxopts = append(currsoxopts, "sinc")

	if bandhigh >= 22050 {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow))
	} else {
		currsoxopts = append(currsoxopts, strconv.Uitoa(bandlow) + "-" + strconv.Uitoa(bandhigh))
	}

	currsoxopts = append(currsoxopts, []string{"channels", "2"})

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
		for {
			frame := new(frame)
			var tmp int16
			err := binary.Read(p.Stdout, binary.BigEndian, &tmp)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					break
				}
			}
			frame.left += float64(tmp)
			err = binary.Read(p.Stdout, binary.BigEndian, &tmp)
			if err != nil {
				if err != os.EOF {
					panic(err)
				} else {
					break 
				}
			}
			frame.right += float64(tmp)
			datachan <- *frame
		}
		fmt.Fprintln(os.Stderr, "Done reading file")
		quitchan <- true
		fmt.Fprintln(os.Stderr, "Quit message sent. Sox out.")
	}()
	return datachan, quitchan
}
