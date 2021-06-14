package util

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RpmApplyPatches(specfile string, destination string) error {
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

func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func CreateRpmBuildStructure(output string) error {
	if output == "" {
		return errors.New("CreateRpmBuildStructure: File not found")
	}

	if err := os.Mkdir(filepath.Join(output, "SOURCES"), 0700); err != nil {
		return errors.New("")
	}

	return nil
}
