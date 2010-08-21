foreach my $num ((1,2,4,6)) {
    if (log(${num})/log(2) != int(log($num)/log(2))) {
        warn;
    } else {
        print "ok\n";
    }
}