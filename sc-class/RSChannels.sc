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
	var <channels, <bandcount, <filename, ready, bufArray;
	*newWithFilename { 
		arg f;
		^super.newCopyArgs(nil, nil, f, false);
	}
	
	read {
		if (this.ready, { ^this } );
		try {
			var file = File.open(this.filename, "rb");
			var nextChar, beats, done, beatcount;
			channels = Dictionary.new;
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
				channels = channels.add(inputname -> beats);
		 		nextChar = file.getChar();
		 		if ( nextChar == nil, {done = true});
			});
		} {|error|
			error.throw
		};
		ready = true;
	}
	
	setupServer {
		|server, bufs, frames, chans|
		if (server.options.numBuffers() < bufs, {server.options.numBuffers_(bufs); server.quit; server.boot});
		bufArray = Buffer.allocConsecutive(bufs, server, frames, chans);
	}
		
	mkSynthDef {
		arg server, name;
		var maxbeats, bufsPerSet; // init
		if (ready == false,{this.read});
		maxbeats = 0;
		channels.keysValuesDo({|key, item|
			if (item.size > maxbeats, {maxbeats = item.size});
		});
		bufsPerSet = bandcount * maxbeats;
		setupServer(server, bufsPerSet);
//		SynthDef(name, { 
	}	
}