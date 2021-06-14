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

	r, err := rpmtools.RpmGetSrcRpm("http://ftp.iinet.net.au/pub/fedora/linux/updates/34/Modular/SRPMS/Packages/c/cri-o-1.20.0-1.module_f34+10489+4277ba4d.src.rpm", "/tmp/")

	if err != nil {
		fmt.Printf(err.Error())
	}

	fmt.Printf(r.RpmGetSource0())

	if err := r.RpmApplyPatches(); err != nil {
		fmt.Println(err.Error())
	}

}
