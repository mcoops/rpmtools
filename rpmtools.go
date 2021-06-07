package rpmtools

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type RpmSpec struct {
	location string
	Tags     map[string][]SpecTag
}

type SpecTag struct {
	TagName  string
	TagValue string
}

var SpecfileLabelsRegex map[string]*regexp.Regexp

func init() {
	if _, err := exec.LookPath("rpmbuild"); err != nil {
		log.Fatal("rpmbuild is required in PATH")
	}

	if _, err := exec.LookPath("rpmspec"); err != nil {
		log.Fatal("rpmspec is required in PATH")
	}

	// https://github.com/bkircher/python-rpm-spec/blob/master/pyrpm/spec.py
	SpecfileLabelsRegex = make(map[string]*regexp.Regexp)
	SpecfileLabelsRegex["name"] = regexp.MustCompile("^Name\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["version"] = regexp.MustCompile("^Version\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["epoch"] = regexp.MustCompile("^Epoch\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["release"] = regexp.MustCompile("^Release\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["summary"] = regexp.MustCompile("^Summary\\s*:\\s*(.+)")
	SpecfileLabelsRegex["license"] = regexp.MustCompile("^License\\s*:\\s*(.+)")
	SpecfileLabelsRegex["url"] = regexp.MustCompile("^URL\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["buildroot"] = regexp.MustCompile("^BuildRoot\\s*:\\s*(\\S+)")
	SpecfileLabelsRegex["buildarch"] = regexp.MustCompile("^BuildArch\\s*:\\s*(\\S+)")

	SpecfileLabelsRegex["sources"] = regexp.MustCompile("^(Source\\d*\\s*):\\s*(.+)")
	SpecfileLabelsRegex["patches"] = regexp.MustCompile("^(Patch\\d*\\s*):\\s*(\\S+)")
	SpecfileLabelsRegex["requires"] = regexp.MustCompile("^Requires\\s*:\\s*(.+)")
	SpecfileLabelsRegex["conflicts"] = regexp.MustCompile("^Conflicts\\s*:\\s*(.+)")
	SpecfileLabelsRegex["obsoletes"] = regexp.MustCompile("^Obsoletes\\s*:\\s*(.+)")
	SpecfileLabelsRegex["provides"] = regexp.MustCompile("^Provides\\s*:\\s*(.+)")
	SpecfileLabelsRegex["packages"] = regexp.MustCompile("^%package\\s+(\\S+)")
}

func rpmFindSpec(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", errors.New("Cannot scan dir for specfile: " + dir)
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".spec" {
			return filepath.Join(dir, f.Name()), nil
		}
	}
	return "", errors.New("specfile not found")
}

func RpmFindAndParseSpec(dir string) (string, RpmSpec, error) {
	specfile, err := rpmFindSpec(dir)
	if err != nil {
		return "", RpmSpec{}, err
	}

	spec, err := RpmParseSpec(specfile)
	return specfile, spec, err
}

func RpmParseSpec(name string) (RpmSpec, error) {
	rpm := RpmSpec{
		Tags: make(map[string][]SpecTag),
	}
	rpm.location = name
	// run rpmspec first to normalize the data
	out, err := exec.Command("rpmspec", "-P", name).Output()
	if err != nil {
		return RpmSpec{}, err
	}

	sc := bufio.NewScanner(bytes.NewReader(out))

	for sc.Scan() {
		for k, i := range SpecfileLabelsRegex {
			if match := i.FindStringSubmatch(sc.Text()); match != nil {
				if len(match) == 2 {
					rpm.Tags[k] = append(rpm.Tags[k], SpecTag{TagName: k, TagValue: match[1]})
				} else if len(match) == 3 {
					rpm.Tags[k] = append(rpm.Tags[k], SpecTag{TagName: match[1], TagValue: match[2]})
				}
			}
		}
	}
	return rpm, nil
}

func (rpm RpmSpec) RpmGetSource0() (string, error) {
	if rpm.Tags["sources"] == nil {
		return "", errors.New("No sources")
	}

	for _, source := range rpm.Tags["sources"] {
		switch source.TagName {
		case "Source0":
			fallthrough
		case "Source":
			return source.TagValue, nil
		}
	}

	// don't find any? we can only assume the first value
	return rpm.Tags["sources"][0].TagValue, nil
}

func (rpm RpmSpec) RpmApplyPatches(destination string) error {
	if strings.HasSuffix(destination, "SOURCES") {
		destination = strings.Replace(destination, "SOURCES", "", 1)
	}
	cmd := exec.Command("bash", "-c", "rpmbuild --nodeps --define \"_topdir "+destination+" \" -bp "+rpm.location)

	if err := cmd.Run(); err != nil {
		// log.Printf("rpmbuild failed to apply patches: %s %s", destination, err)
		return err
	}

	return nil
}
