package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mcoops/rpmtools"
)

func main() {
	urlPtr := flag.String("srpm", "", "url/filepath to download and patch specfile, i.e. http://ftp.iinet.net.au/pub/fedora/linux/updates/34/Modular/SRPMS/Packages/c/cri-o-1.20.0-1.module_f34+10489+4277ba4d.src.rpm")
	noCleanup := flag.Bool("nocleanup", false, "does not clean up the RPM directories at the end")

	flag.Parse()

	if *urlPtr == "" {
		fmt.Println("Must supply a URL to download srpm")
		return
	}

	var filePath string
	if strings.HasPrefix(*urlPtr, "file://") {
		filePath = (*urlPtr)[len("file://"):]
	} else if strings.HasPrefix(*urlPtr, "http://") || strings.HasPrefix(*urlPtr, "https://") {
		resp, err := http.Get(*urlPtr)
		if err != nil {
			fmt.Println("Could not download the srpm")
			return
		}
		defer resp.Body.Close()

		out, err := ioutil.TempFile("", "")
		if err != nil {
			fmt.Println("Could not download the srpm")
			return
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Println("Could not download the srpm")
			return
		}
		filePath = out.Name()
	} else {
		fmt.Println("srpm flag should start with file:// or http(s)://")
		return
	}

	r, err := rpmtools.RpmSpecFromFile(filePath, "/tmp")
	if err != nil {
		fmt.Printf("%s", err.Error())
	}

	source0, _ := r.GetSource0()
	fmt.Println("Source0: " + source0)
	fmt.Printf("Licenses: ")
	fmt.Println(r.Tags["license"])
	fmt.Println("SpecLocation: " + r.SpecLocation)
	fmt.Println("SRPMLocation: " + r.SrpmLocation)
	fmt.Println("SourcesLocation: " + r.SourcesLocation)
	fmt.Println("OutLocation: " + r.OutLocation)
	fmt.Println("BuildLocation: " + r.BuildLocation)

	for name, tag := range r.Tags {
		fmt.Printf("Tag %s = %s\n", name, tag)
	}

	if err := r.ApplyPatches(); err != nil {
		fmt.Println(err.Error())
	}

	if !*noCleanup {
		r.Cleanup()
	}
}
