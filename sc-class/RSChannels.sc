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
	var <channels, <bandcount, <filename, ready, <>bps;
	*newWithFilename { 
		arg f;
		^super.newCopyArgs(nil, nil, f, false, 0.25);
	}
	
	read {
		if (ready, { ^this } );
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
		arg server;
		fork {
		var maxbeats, bufsPerSet, t, recorderInput, playInput, recoffset=0, playoffset=0, recbeat=0, playbeat=0, firstplay=true, fullmatrix, currbeat=0, bufArray; // init
		if (ready == false,{this.read});
		maxbeats = 0;
		channels.keysValuesDo({|key, item|
			if (item.size > maxbeats, {maxbeats = item.size});
		});
		bufsPerSet = bandcount * maxbeats;
		if (server.options.numBuffers() < (bufsPerSet * 4), {
			var booted = Condition.new;
			server.options.numBuffers_(bufsPerSet * 4);
			server.reboot;
			10.sleep;
			"server up".postln;
		});
		// min / beat * 60 s / min * 44100 frames / sec
		bufArray = Buffer.allocConsecutive(bufsPerSet * 2, server, 1.0 * 44100.0 / bps, 2);
		(1.0 * 44100.0 / bps).postln;
		bufArray[0].postln;
		bufArray[0].loadToFloatArray(action: {
					|array|
					"moo".postln;
					array.postln;
/*					var leftsum = 0.0, rightsum = 0.0;
					array.do({|item, i| if (i.even, {leftsum = leftsum + item}, {rightsum = rightsum + item}); });
					
					("bufarray" + j + "left" + leftsum + "right" + rightsum ).postln;
					if (leftsum >= recorderInput[1][recbeat].bands[j].left, {
						if (rightsum >= recorderInput[1][recbeat].bands[j].right, {fullmatrix[recbeat][j] = true});
					});
					i = i - 1;
					if (i == 0, {c.test = true});
					c.signal;
					"loadtofloatarray complete".postln;*/
				});
		recoffset = playoffset = bufArray[0].bufnum;
		
		// set up recorders
		// choose an input to record
		recorderInput = channels.asSortedArray.choose;
		fullmatrix = Array.fill2D(recorderInput[1].size, bandcount, false);
		// SynthDef for recorders
		SynthDef(\RSChannelsRecord, {
			arg bufnum = 0, i = 0;
			RecordBuf.ar(BPF.ar(In.ar(0,2), (22050.0/bandcount)*(i+0.5), (1.0*(i+1)/bandcount), bufnum), 0, 1.0, 1.0, doneAction:2);
		}).send(server);
		t = TempoClock(bps);
		// schedule a new recorder set every beat
		t.sched(1.0, { var c, i = bandcount, blockreturn;
			bandcount.do({
				|index|
				("synth construct" + 
				(bufArray[0].bufnum+recoffset+(recbeat*bandcount)+index) + (recoffset+(recbeat*bandcount)+index)).postln;
				if (fullmatrix[recbeat][index] == false, {
					Synth(\RSChannelsRecord, [\bufnum, bufArray[0].bufnum+recoffset+(recbeat*bandcount)+index, \i, recoffset+(recbeat*bandcount)+index]);
				});
				"synth construct complete".postln;
			});
			c = Condition.new(false);
			bandcount.do({
				|j|
				var b = bufArray.at(recoffset+(recbeat*bandcount)+j);
				bufArray.postln;
				(recoffset+(recbeat*bandcount)+j).postln;
				b.query;
				b.loadToFloatArray(action: {
					|array|
					array.postln;
/*					var leftsum = 0.0, rightsum = 0.0;
					array.do({|item, i| if (i.even, {leftsum = leftsum + item}, {rightsum = rightsum + item}); });
					
					("bufarray" + j + "left" + leftsum + "right" + rightsum ).postln;
					if (leftsum >= recorderInput[1][recbeat].bands[j].left, {
						if (rightsum >= recorderInput[1][recbeat].bands[j].right, {fullmatrix[recbeat][j] = true});
					});
					i = i - 1;
					if (i == 0, {c.test = true});
					c.signal;
					"loadtofloatarray complete".postln;*/
				});
				j.postln;
			});
			Routine.new({
				"waiting".postln;
				c.wait; 
				"wait complete".postln;
				blockreturn = block({
					|break|
					fullmatrix.do({
						|m|
						m.do({
							|n|
							if (n == false, {
								break.value(false)
							})
						})
					});
				});
			}).value; // wtf?
			if (blockreturn != false, { 
				var tmp = recoffset;
				recbeat = playbeat = 0;
				if (firstplay, {
					recoffset = recoffset + bufsPerSet;
					firstplay = false;
				}, {
					recoffset = playoffset;
					playoffset = tmp;
				});
				playInput = recorderInput;
				recorderInput = channels.asSortedArray.choose;
			}, {
				recbeat = (recbeat + 1) % recorderInput[1].beats.size
			});
		1.0 });
		// set up players
//		SynthDef(\RSChannelsPlay, { |offset, beat| Mix.fill( bandcount, { | i | PlayBuf.ar(2, offset+(beat*bandcount)+i, BufRateScale.kr(offset+(beat*bandcount)+i), doneAction:2)})}).load(server);
		// schedule new synth to play every beat increasing offset
//		t.sched(1.0, {
//			Synth(\RSChannels, [\offset, playoffset, \beat, playbeat]);
//			playbeat = (playbeat + 1) % playInput[1].beats.size;
//			1.0;
//		});
	}	
	}

//	destroy {
//	}
}