package util

import (
	"errors"
	"io"
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

func createDir(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.New("CreateRpmBuildStructure: failed to create dir " + path + " - " + err.Error())
	}
	return nil
}

func CreateRpmBuildStructure(output string) (string, string, string, error) {
	if output == "" {
		return "", "", "", errors.New("CreateRpmBuildStructure: no file specified")
	}

	sourceRPM := filepath.Join(output, "SOURCES")
	if err := createDir(sourceRPM); err != nil {
		return "", "", "", err
	}

	sRPM := filepath.Join(output, "SRPMS")
	if err := createDir(sRPM); err != nil {
		return sourceRPM, "", "", err
	}

	bRPM := filepath.Join(output, "BUILD")
	if err := createDir(bRPM); err != nil {
		return sourceRPM, sRPM, "", err
	}

	return sourceRPM, sRPM, bRPM, nil
}

func DirEmpty(loc string) error {
	f, err := os.Open(loc)
	if err != nil {
		return errors.New("ApplyPatches: failed to open output location: " + loc)
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return errors.New("ApplyPatches: dir is empty: " + loc)
	}

	return nil
}
