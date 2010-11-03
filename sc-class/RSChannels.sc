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
	var <channels, <bandcount, <filename, ready, bufArray, synth, <>bps;
	*newWithFilename { 
		arg f;
		^super.newCopyArgs(nil, nil, f, false, nil, nil, 64.0);
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

	prepare {
		arg server, name;
		var maxbeats, bufsPerSet, t, recorderInput; // init
		if (ready == false,{this.read});
		maxbeats = 0;
		channels.keysValuesDo({|key, item|
			if (item.size > maxbeats, {maxbeats = item.size});
		});
		bufsPerSet = bandcount * maxbeats;
		if (server.options.numBuffers() < (bufsPerSet * 4), {server.options.numBuffers_(bufsPerSet * 4); server.quit; server.boot});
		// min / beat * 60 s / min * 44100 frames / sec
		bufArray = Buffer.allocConsecutive(bufsPerSet * 2, server, 1.0 * 44100.0 / bps, 2);
		
		// set up recorders
		// choose an input to record
		recorderInput = channels.asSortedArray.choose;
		SynthDef(\RSChannelsRecord, { |offset| Mix.fill( bandcount, { | i | RecordBuf.ar(BPF.ar(In.ar(0,2), (22050.0/bandcount)*(i+0.5), (1.0*i/bandcount)), offset+i, 0, 1.0, 1.0, doneAction:2)})}).load(server);
		
		// set up players
		SynthDef(\RSChannelsPlay, { |offset| Mix.fill( bandcount, { | i | PlayBuf.ar(2, offset+i, BufRateScale.kr(offset+1), doneAction:2)})}).load(server);
		// schedule new synth to play every beat increasing offset
		t = TempoClock(bps);
//		t.sched(1.0, {
//			Synth(\RSChannels, 
	}	

	destroy {
	}
}