#!/usr/bin/env perl -w

use strict;

use Getopt::Long;
use File::Basename;
use Storable;

# config section
my $bands = 20;

my $basedir = dirname $0;
my $tempdir = "$basedir/bits";
mkdir $tempdir;

my %tracks;
my $tracklist = shift;
open(my $tracklistfh, "<", $tracklist) or die $!; 
my $in = 0;
my ($filename, $length);
while (<$tracklistfh>) {
    if (/^BEGIN (.*)$/) {
        $filename = $1;
        $in = 1;
    } elsif (/^END$/) {
        $in = 0;
        my $hash = `md5 -q $filename`;
        chomp $hash;
        $tracks{$filename} = { length => $length, hash => $hash };
    } elsif (/^LENGTH (.*)$/) {
        if ($1 =~ /[^0-9]/) { warn "bad length"; }
        $length = $1;
    }
}
close $tracklistfh;

use Data::Dumper;

my $bandwidth = 22050 / $bands;
foreach my $trackfile (keys %tracks) {
    if ( -f "$basedir/bits/stats/".$tracks{$trackfile}{hash} ) {
        $tracks{$trackfile}{stats} = retrieve "$basedir/bits/stats/".$tracks{$trackfile}{hash};
        next;
    }
    my $trackinfo = `soxi '$trackfile'`;
    $trackinfo =~ /^Duration.* (\d+) samples /m;
    my $samples = $1;
    my $beatlength = $samples / $tracks{$trackfile}{length};
    my %trackstats = ();
    $trackstats{beatlength} = $beatlength;
    $trackinfo =~ /^Channels.* (\d+)/m;
    my $channels = $1;
    my $remixspec = "";
    if ($channels == 1) {
        $remixspec = "remix 1 1";
    }
    my $band = 0;
    for (my ($bandlow, $bandhigh) = (0, $bandwidth); $bandhigh <= 22050; $bandlow += $bandwidth, $bandhigh += $bandwidth) {
        my $sox = soxcmd($trackfile, $remixspec, $bandlow, $bandhigh);
        print "$cmd\n";
        open my ($sox), $cmd;
        my ($bytesread, $buf);
        my $beat = 0;
        my $samplesinbeat = 0;
        my $channel = 0;
        while ($bytesread = read($sox, $buf, 8192)) {
            my @samples = unpack("s*", $buf);
            foreach my $sample (@samples) {
                $trackstats{grid}[$band][$channel][$beat] += abs($sample);
                ++$samplesinbeat;
                if ($samplesinbeat >= $beatlength && $beat < $tracks{$trackfile}{length} - 1) {
                    ++$beat;
                    $samplesinbeat = 0;
                }
                $channel = !$channel;
            }
        }
        close $sox;
        $band++;
    }
    store \%trackstats,"$basedir/bits/stats/".$tracks{$trackfile}{hash};
    $tracks{$trackfile}{stats} = \%trackstats;
}

my $inputdir = shift;
opendir my ($inputdirfh), $inputdir;
my ($trackfile, $currenttrack) = each %tracks;
my %inputfiles;
my @output;
my @outputstats;
my $completedbuckets = 0;
my $completiontarget = $bands * $currenttrack->{length};
foreach my $inputfile (readdir($inputdirfh)) {
    next if !-f "$inputdir/$inputfile";
    $inputfiles{"$inputdir/$inputfile"} = 1;
} # we want random order!

print Dumper(\%inputfiles);

my $inputfile = each %inputfiles
my $inputinfo = `soxi '$inputfile'`;
$inputinfo =~ /^Channels.* (\d+)/m;
my $channels = $1;
my $remixspec = "";
if ($channels == 1) {
    $remixspec = "remix 1 1";
}

my $sox = soxcmd($inputfile, $remixspec, $bandlow, $bandhigh);
my $beat = 0;
my $samplesinbeat = 0;
my $channel = 0;
my $sampledex = 0;

OUTER: while (1) {
    my $band = 0;
    BAND: for (my ($bandlow, $bandhigh) = (0, $bandwidth); $bandhigh <= 22050; $bandlow += $bandwidth, $bandhigh += $bandwidth) {
        my ($bytesread, $buf);
        SAMPLEBUF: while ($bytesread = read($sox, $buf, 8192)) {
            my @samples = unpack("s*", $buf);
            
            SAMPLE: while ($sampledex < 8192) {
                if ($outputstats[$band][$channel][$beat] < $currenttrack->{stats}{grid}[$band][$channel][$beat]) {
                    if ($beat < $currenttrack->{length}) {
                        ++$beat;
                    } else {
                        next BAND;
                $outputstats[$band][$channel][$beat] += $samples[$sampledex];
                && $samplesinbeat < $beatlength && 
                
        }
        close $sox;
        $band++;
    }
}

sub output {
    my ($trackfile, $output) = @_;
 #   print Dumper($output);
}

sub soxcmd {
    my ($inputfile, $remixspec, $bandlow, $bandhigh) = @ARGV;
    my $pass = "-$bandhigh";
    if ($bandhigh == 22050) { 
        $pass = "";
    }
    my $cmd = "sox '$inputfile' -b 16 -e signed-integer -r 44100 -t raw - $remixspec sinc $bandlow$pass channels 2 |";
    print $cmd;
    open my ($fh), $cmd;
    return $fh;
}