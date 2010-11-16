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
)

type source struct {
	filename string
	beats []remixoscope.Beat
}

type ByteArrayReader struct {
	source []byte
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
			s.beats = append(s.beats, remixoscope.Beat{make([]remixoscope.Frame, 0)})
			for j := uint(0); j < bandcount; j++ {
				s.beats[i].Frames = append(s.beats[i].Frames, remixoscope.Frame{})
				var out uint64
				binary.Read(bar, binary.BigEndian, &out)
				s.beats[i].Frames[j].Left = math.Float64frombits(out)
				binary.Read(bar, binary.BigEndian, &out)
				s.beats[i].Frames[j].Right = math.Float64frombits(out)
			}
		}
		(*inputs)[infile] = s
	}
	return bandcount, inputs
}

func main() {
	var f remixoscope.Soxsample = 4
	fmt.Printf("%d\n", f)
}
