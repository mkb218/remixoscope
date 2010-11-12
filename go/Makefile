include $(GOROOT)/src/Make.inc

all: analyzer producer

include $(GOROOT)/src/Make.common

analyzer: remixoscope.a analyzer.$O
	$(LD) -L . -o analyzer analyzer.$O

producer: remixoscope.a producer.$O
	$(LD) -L . -o producer producer.$O

remixoscope.a: remixoscope.$O
	gopack grc $@ $<

%.$O: %.go
	$(GC) -I . $<
	

CLEANFILES+=analyzer producer