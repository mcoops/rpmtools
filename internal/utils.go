package util

import (
	"os/exec"
	"strings"
)

func rpmApplyPatches(specfile string, destination string) error {
	if strings.HasSuffix(destination, "SOURCES") {
		destination = strings.Replace(destination, "SOURCES", "", 1)
	}
	cmd := exec.Command("bash", "-c", "rpmbuild --nodeps --define \"_topdir "+destination+" \" -bp "+specfile)

	if err := cmd.Run(); err != nil {
		// log.Printf("rpmbuild failed to apply patches: %s %s", destination, err)
		return err
	}

	return nil
}
