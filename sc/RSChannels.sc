BandInfo {
	var <>left, <>right; // floats
	*new {
		left = right = 0.0;
		^super.new
	}
}

BeatInfo {
	var <>bands;
	*new {
		arg size;
		^super.newCopyArgs(Array.new(size));
	}
}



RSChannels[slot] {
	var <channels;
	*newWithFilename { 
		arg filename;
		j = this.new;
		j.read(filename)
	}
	
	read {
		arg filename;
		var file = File.open(filename, "rb");
		if ( !file.isOpen, { Error("couldn't open file" + filename + "!").throw } );
		var nextChar = file.getChar();
		var bandcount = "";
		while ( { nextChar != $| && nextChar != nil }, { bandcount = bandcount ++ nextChar; nextChar = file.getChar() } );
		if (nextChar == nil, { Error("premature eof in" + filename).throw });
		bandcount = bandcount.asInteger;
		var done = false;
		nextChar = file.getChar();
		var beats;
		while ( { done == false }, {
			var inputname;
			while ( { nextChar != $| && nextChar != nil }, { inputname = inputname ++ nextChar; nextChar = file.getChar() } );
			if (nextChar == nil, { Error("premature eof in" + filename).throw });
			nextChar = file.getChar();
			var beatcount = "";
			while ( { nextChar != $| && nextChar != nil }, { beatcount = beatcount ++ nextChar; nextChar = file.getChar() } );
			if (nextChar == nil, { Error("premature eof in" + filename).throw });
			beatcount = beatcount.asInteger;
			beats = Array.new(beatcount)
			beatcount.do({
				arg beat;
				beats[beat] = BeatInfo.new(bandcount);
				bandcount.do({
					arg band;
					beats[beat].bands[band] = BandInfo.new();
					beats[beat].bands[band].left = file.getDouble();
					beats[beat].bands[band].right = file.getDouble();
				});
			});
	 		nextChar = file.getChar();
	 		if ( nextChar == nil, {done = true});
		});
		^beats;
	}
	