BandInfo {
	var <>left, <>right; // floats
}

BeatInfo {
	var <>bands;
	*new {
		super.new;
		bands = Dictionary.new;
	}
}



RSChannels[slot] {
	var <channels;
	*newWithFilename { 
		arg filename;
		j = this.new;
		j.jsoninit(filename);
	}

	jsoninit {
		arg filename;
		super.init;
		var file = File.new(filename, "r");
		this.jsonparse(file);
	}
	
	eatwhitespaceuntil {
		arg file, needle;
		var c = file.getChar();
		while({ c != nil && c.isSpace }, { c = file.getChar() } );
		if (c != needle, {
			Error("never found needle").throw;
		});
	}
	
	eatwhitespaceuntileither {
		arg file, needles;
		needles = "[" ++ needles ++ "]"
		var c = file.getChar();
		while({ c != nil && c.isSpace() }, { c = file.getChar() } );
		if (!String.with(c).matchRegexp(needles), {
			Error("never found needle").throw;
		},);
		^c
	}
	
	collectuntil {
		arg file, needle;
		var s = "";
		var c = file.getChar();
		while({ c != needle }, { c = file.getChar(); s = s ++ c } );
		^c;
	}

	
	jsonparse {
		arg file;
		var out = Dictionary.new()
		var done = false
		while ({!done}, {
			// eat whitespace until i see a leftbrace
			eatwhitespaceuntil(${);
			// eat {, then eat whitespace until i see a "
			eatwhitespaceuntil($");
			// read until next unescaped " and store as filename
			var filename = collectuntil($");
			// eat ", then whitespace until :
			eatwhitespaceuntil($:);
			out[filename] = jsonparsebeats(file);
			// eat whitespace until ,
			var answer = eatwhitespaceuntileither(",}");
			if ((answer == $}), {
				done = true;
			} );
		});
		// or }
		// done
		// return Dictionary
		^out
	}
	
	jsonparsebeats {
		arg file; 
		// eat whitespace until leftbrace
		eatwhitespaceuntil(${);
		// eat whitespace until unescaped "beats"
		
		// eat whitespace until :
		jsonparsebeat(file);
		// eat whitespace until }
		// eat whitespace until either ,
		// parsebeat again or
		// or ]
		// return! done with beats!
	}
	
	jsonparsebeat(file) {
		arg file;
		// eat whitespace until leftbracket
		// eat whitespace until leftbrace
		// eat whitespace until "frames"
		// eat whitespace until :
		// eat whitespace until leftbracket
		frame = jsonparseband(file)
		// eat whitespace until either
		// , -> parse another frame
		// rightbracket return! done with beat
	}
	
	jsonparseband(file) {
		arg file;
		out = FrameInfo.new;
		// eat whitespace until leftbracket
		// DO:
		// eat whitespace until "
		// read chars until "
		// eat whitespace until :
		// eat whitespace until digit
		// parse float store in field named by string
		// UNTIL SEE ,
		// eat whitespace until rightbrace
		return out;
	} 
		
	
}