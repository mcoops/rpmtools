package main

import (
	"flag"
	"fmt"

	"github.com/mcoops/rpmtools"
)

func main() {
	urlPtr := flag.String("srpm", "", "url to download and patch specfile, i.e. http://ftp.iinet.net.au/pub/fedora/linux/updates/34/Modular/SRPMS/Packages/c/cri-o-1.20.0-1.module_f34+10489+4277ba4d.src.rpm")

	flag.Parse()

	if *urlPtr == "" {
		fmt.Println("Must supply a URL to download srpm")
		return
	}

	r, err := rpmtools.RpmGetSrcRpm(*urlPtr, "/tmp/")

	if err != nil {
		fmt.Printf("%s", err.Error())
	}

	source0, _ := r.RpmGetSource0()
	fmt.Println("Source0: " + source0)
	fmt.Printf("Licenses: ")
	fmt.Println(r.Tags["license"])

	if err := r.RpmApplyPatches(); err != nil {
		fmt.Println(err.Error())
	}

	r.RpmCleanup()
}
