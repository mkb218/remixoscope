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
)

type source struct {
	filename string
	beats []remixoscope.Beat
}

type ByteArrayReader struct {
	source []byte
}

type AudioBuffer struct {
	left []int16
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
	for ; !done; {
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
	datachan := make(chan AudioBuffer)
	quitchanout := make(chan bool)
	go func() {
		for ; len(files) > 0; {
			filename := files[0]
			files = files[1:]
			soxopts := []string{"sox", filename, "-b", "16", "-c", "2", "-e", "signed-integer",  "-B", "-r", "44100", "-t", "raw", "-", "sinc"}
			bandwidth := 22050 / bands
			bandlow := band * bandwidth
			bandhigh := bandlow + bandwidth
	
			if bandhigh >= 22050 {
				soxopts = append(soxopts, strconv.Uitoa(bandlow))
			} else {
				soxopts = append(soxopts, strconv.Uitoa(bandlow) + "-" + strconv.Uitoa(bandhigh))
			}

			rawchan, quitchanin := remixoscope.Startsox(sox, soxopts)
			for {
				ab := AudioBuffer{make([]int16, length), make([]int16, length)}
				for i := uint(0); i < length; i++ {
					f := <- rawchan
					ab.left[i] = f.Left
					ab.right[i] = f.Right
				}
				datachan <- ab
			}
		}
	}
		
	return datachan, quitchanout
}

func main() {
	output := flag.String("analysis", "analysis", "output of analyzer run")
	soxpath, _ := exec.LookPath("sox")
	sox := flag.String("sox", soxpath, "Path to sox binary. Default is to search path")
	outputfile := flag.String("output", "output.wav", "mixed output")
}
