BandInfo {
	var <>left, <>right; // floats
	*new {
		^super.newCopyArgs(0.0, 0.0)
	}
}

BeatInfo {
	var <>bands;
	*new {
		arg size;
		^super.newCopyArgs(Array.new(size));
	}
}

RSChannels {
	var <channels, <beatcount, <>filename;
	*newWithFilename { 
		arg f;
		var o = super.new;
		o.filename = f;
		^o;
	}
	
	read {
		try {
			var file = File.open(this.filename, "rb");
			var nextChar, beats, done, bandcount;
			"file open".postln;
			if ( file.isOpen != true, { Error("couldn't open file" + filename + "!").throw } );
			nextChar = file.getChar();
			bandcount = "";
			while ( { (nextChar != $|).and(nextChar != nil) }, { bandcount = bandcount ++ nextChar; nextChar = file.getChar(); } );
			bandcount = bandcount.asInteger;
			if (nextChar == nil, { Error("never found | after bandcount. premature eof in" + filename).throw });
			done = False;
			nextChar = file.getChar();
			while ( { done == False }, {
				var inputname;
				while ( { (nextChar != $|).and(nextChar != nil) }, { inputname = inputname ++ nextChar; nextChar = file.getChar() } );
				if (nextChar == nil, { Error("premature eof in" + filename).throw });
				nextChar = file.getChar();
				beatcount = "";
				while ( { (nextChar != $|).and(nextChar != nil) }, { beatcount = beatcount ++ nextChar; nextChar = file.getChar() } );
				if (nextChar == nil, { Error("premature eof in" + filename).throw });
				beatcount = beatcount.asInteger;
				beats = Array.new(beatcount);
				beatcount.do({
					arg beat;
					beats = beats.add(BeatInfo.new(bandcount));					bandcount.do({
						arg band;
						beats[beat].bands.add(BandInfo.new());
						beats[beat].bands[band].left = file.getDouble();
						beats[beat].bands[band].right = file.getDouble();
					});
				});
		 		nextChar = file.getChar();
		 		if ( nextChar == nil, {done = true});
			});
			channels = beats;
		} {|error|
			error.throw
		};
	}
}