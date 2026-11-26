package main

import (
	"cmp"
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

func main() {
	var stats unix.Statfs_t
	if err := unix.Statfs("/volume", &stats); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d %d %d\n", cmp.Or(stats.Frsize, stats.Bsize), stats.Blocks, stats.Bfree)
}
