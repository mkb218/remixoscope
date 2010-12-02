package main

import (
	"remixoscope"
	"os"
	"fmt"
	"math"
	ioutil "io/ioutil"
	"strconv"
	"strings"
	binary "encoding/binary"
	"flag"
	"rand"
	"syscall"
)

type source struct {
	filename string
	beats    []remixoscope.Beat
}

type ByteArrayReader struct {
	source []byte
}

type AudioBuffer struct {
	left  []int16
	right []int16
}

func (s *ByteArrayReader) Read(p []byte) (n int, err os.Error) {
	max := len(p)
	c := len(s.source)
	err = nil
	if c >= max {
		for i, v := range s.source[0:max] {
			p[i] = v
		}
		n = max
	} else {
		for i, v := range s.source[0:c] {
			p[i] = v
		}
		n = c
		err = os.EOF
	}
	s.source = s.source[n:]
	return n, err
}

func inputMapFromOutput(filename string) (bandcount uint, inputs *map[string]source) {
	inputs = new(map[string]source)
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	b := strings.Index(string(contents), "|")
	bandcount, err = strconv.Atoui(string(contents[0:strings.Index(string(contents), "|")]))
	if err != nil {
		panic(err)
	}
	contents = contents[b+1:]
	done := false
	var s source
	for !done {
		b = strings.Index(string(contents), "|")
		infile := string(contents[0:b])
		contents = contents[b+1:]

		b = strings.Index(string(contents), "|")
		var beatcount uint
		beatcount, err = strconv.Atoui(string(contents[0:b]))
		contents = contents[b+1:]

		s.beats = make([]remixoscope.Beat, 0)

		bar := new(ByteArrayReader)
		bar.source = contents
		for i := uint(0); i < beatcount; i++ {
			s.beats = append(s.beats, remixoscope.Beat{make([]remixoscope.Bucket, 0)})
			for j := uint(0); j < bandcount; j++ {
				s.beats[i].Buckets = append(s.beats[i].Buckets, remixoscope.Bucket{})
				var out uint64
				binary.Read(bar, binary.BigEndian, &out)
				s.beats[i].Buckets[j].Left = math.Float64frombits(out)
				binary.Read(bar, binary.BigEndian, &out)
				s.beats[i].Buckets[j].Right = math.Float64frombits(out)
			}
		}
		(*inputs)[infile] = s
	}
	return bandcount, inputs
}

func openband(sox string, files []string, length uint, band uint, bands uint) (chan AudioBuffer, chan bool) {
	interchan := make(chan AudioBuffer, 5)
	quitchanout := make(chan bool)
	go func() {
		for i := 0; ; i = (i + 1) % len(files) {
			filename := files[i]
			filelength, channels := remixoscope.Getfileinfo(sox, filename)

			rawchan, quitchanin := remixoscope.Openband(sox, channels, filename, band)
			for !closed(rawchan) {
				interchan <- rawchan
			}

		}
		close(interchan)
		close(quitchanout)
	}()

	go func() {
		for !closed(interchan) {
			ab := AudioBuffer{make([]int16, length), make([]int16, length)}
			for i := uint(0); i < length; i++ {
				f := <-interchan
				ab.left[i] = f.Left
				ab.right[i] = f.Right
			}
			datachan <- ab
		}
	}()

	return datachan, quitchanout
}

func shuffle(filenames []string) {
	sv := StringVector(filenames)
	for i := len(filenames) - 1; i >= 1; i-- {
		j := rand.Intn(i)
		sv.Swap(i, j)
	}
}

func main() {
	output := flag.String("analysis", "analysis", "output of analyzer run")
	tmpdir := flag.String("tmpdir", "/tmp", "tmpdir")
	soxpath, _ := exec.LookPath("sox")
	sox := flag.String("sox", soxpath, "Path to sox binary. Default is to search path")
	outputfile := flag.String("output", "output.wav", "mixed output")
	beatlength := flag.Uint("beatlength", 0, "length of each beat in sample frames")
	flag.Parse()
	if *beatlength == 0 {
		fmt.Fprintln(os.Stderr, "Bad beatlength")
	}
	bands, loops := inputMapFromOutput(*output)

	for filename, info := range loops {
		fifos := make([]os.File, bands)
		for band := 0; band < bands; band++ {
			fifoname := *tmpdir + "/producer-" + filename[strings.LastIndex(filename, "/")+1:] + strconv.Itoa(band)
			errno := syscall.Mkfifo(fifoname, uint32(0700))
			if errno != 0 {
				panic(os.NewSyscallError("Mkfifo", errno))
			}
			datachan, quitchan := openband(*sox, flag.Args(), *beatlength, band, bands)
			go func() {
				fifo, err := os.Open(fifoname, os.O_WRONLY, 0)
				if err != nil {
					panic(err)
				}
				for {
					var buf AudioBuffer
					for {
						ab := <-datachan
						err = binary.Write(fifo, binary.BigEndian, ab)
						if err != nil {
							panic(err)
						}
					}
				}

			}()
		}
	}
}
