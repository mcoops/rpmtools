package main

import (
	"flag"
	"fmt"

	"github.com/mcoops/rpmtools"
)

func main() {
	repoPtr := flag.String("repo", "", "scan repo for specfile")

	flag.Parse()

	fmt.Println(*repoPtr)

	r, _ := rpmtools.RpmParseSpec("/tmp/cri-o/contrib/test/ci/cri-o.spec")

	fmt.Println(r)

	fmt.Println(r.RpmGetSource0())
}
