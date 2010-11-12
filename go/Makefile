include $(GOROOT)/src/Make.inc

all: analyzer producer

include $(GOROOT)/src/Make.common

analyzer: remixoscope.$O analyzer.$O
	$(LD) -o analyzer remixoscope.$O analyzer.$O

producer: remixoscope.$O producer.$O
	$(LD) -o producer remixoscope.$O analyzer.$O

%.$O: %.go
	$(GC) $<
	

