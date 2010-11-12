package remixoscope

import (
	vector "container/vector"
	"bufio"
	"fmt"
	"os"
);
type soxsample int32
type frame struct {
	left, right float64
}

type beat struct {
	frames []frame // one frame per band
}
type track struct {
	beats []beat
	info *input
}

type config struct {
	inputlist string
	bands     uint
	sox       string
	soxopts   vector.StringVector
	output    *bufio.Writer
}

type input struct {
	beats      uint
	beatlength uint
	channels   uint
}

type StringWriter struct {
	out *string
}

func (this *StringWriter) Write(p []byte) (n int, err os.Error) {
	tmp := fmt.Sprintf("%s", p)
	this.out = &tmp
	return len(tmp), nil
}

