package objabi

import "flag"

func Flagparse(usage func()) {
	flag.Usage = usage
	flag.Parse()
}
