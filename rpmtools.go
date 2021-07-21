package rpmtools

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	util "github.com/mcoops/rpmtools/internal"
)

type RpmSpec struct {
	SpecLocation    string
	SrpmLocation    string
	SourcesLocation string
	OutLocation     string
	BuildLocation   string
	Tags            map[string][]SpecTag
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
	SpecfileLabelsRegex["buildRequires"] = regexp.MustCompile("^BuildRequires\\s*:\\s*(.+)")

	SpecfileLabelsRegex["sources"] = regexp.MustCompile("^(Source\\d*\\s*):\\s*(.+)")
	SpecfileLabelsRegex["patches"] = regexp.MustCompile("^(Patch\\d*\\s*):\\s*(\\S+)")
	SpecfileLabelsRegex["requires"] = regexp.MustCompile("^Requires\\s*:\\s*(.+)")
	SpecfileLabelsRegex["conflicts"] = regexp.MustCompile("^Conflicts\\s*:\\s*(.+)")
	SpecfileLabelsRegex["obsoletes"] = regexp.MustCompile("^Obsoletes\\s*:\\s*(.+)")
	SpecfileLabelsRegex["provides"] = regexp.MustCompile("^Provides\\s*:\\s*(.+)")
	SpecfileLabelsRegex["packages"] = regexp.MustCompile("^%package\\s+(\\S+)")
}

// Find the first file ending with .spec
func RpmFindSpec(dir string) (string, error) {
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

// Using the first specfile found, parse it's fields and return an struct
// allowing easy asccess to fields.
func RpmFindAndParseSpec(dir string) (RpmSpec, error) {
	specfile, err := RpmFindSpec(dir)
	if err != nil {
		return RpmSpec{}, err
	}

	spec, err := RpmParseSpec(specfile)
	return spec, err
}

func rpmCleanSpecFile(name string) error {
	hasChanges := false
	f, err := ioutil.ReadFile(name)

	if err != nil {
		return errors.New("rpmCleanSpecFile: failed to open file " + name)
	}
	lines := strings.Split(string(f), "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			break
		}
		if strings.HasPrefix(line, "%__") {
			lines[i] = "#" + line
			hasChanges = true
		}
	}

	if !hasChanges {
		return nil
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(name, []byte(output), 0644)
	if err != nil {
		return errors.New("rpmCleanSpecFile: failed to writeback file")
	}
	return nil
}

// Given a specfile parse and return fields from the file
func RpmParseSpec(name string) (RpmSpec, error) {
	if util.Exists(name) == false {
		return RpmSpec{}, errors.New("File: " + name + " not found")
	}

	rpmCleanSpecFile(name)

	rpm := RpmSpec{
		Tags: make(map[string][]SpecTag),
	}
	rpm.SpecLocation = name
	// run rpmspec first to normalize the data
	out, err := exec.Command("rpmspec", "-P", name).Output()
	if err != nil {
		// rpmspec will occasionally return errors, so ignore them
		log.Printf("RpmParseSpec: ignoring error %s", err.Error())
		// return RpmSpec{}, err
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

// Using a rpmspec obj return source0
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

// Using an rpmspec obj (rpm.spec location) and an output location, extract
// the source rpm and apply patches
func (rpm RpmSpec) RpmApplyPatches() error {
	if !strings.HasSuffix(rpm.SourcesLocation, "SOURCES") {
		return errors.New("RpmApplyPatches: expected SOURCES path is incorrect: " + rpm.SourcesLocation)
	}
	cmd := exec.Command("bash", "-c", "rpmbuild -bp --nodeps --define \"_topdir "+rpm.OutLocation+" \" "+rpm.SpecLocation)

	if err := cmd.Run(); err != nil {
		return errors.New("RpmApplyPatches: failed to run rpmbuild: " + err.Error())
	}

	return nil
}

// Best effort. Should check each error code, but prob not worth
func (rpm RpmSpec) RpmCleanup() {
	os.RemoveAll(rpm.SourcesLocation)
	os.RemoveAll(rpm.SrpmLocation)
	os.RemoveAll(rpm.BuildLocation)
	os.RemoveAll(filepath.Join(rpm.OutLocation, "BUILDROOT"))
	os.RemoveAll(filepath.Join(rpm.OutLocation, "RPMS"))
}

// Given a `url` download the rpm to the `outputPath` to `SRPM` folder. Then
// using rpm2cpio attempt to unpack to `SOURCES`. If that all completes, find
// and parse the specfile.
func RpmGetSrcRpm(url string, outputPath string) (RpmSpec, error) {
	resp, err := http.Get(url)

	if err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to fetch url: " + url)
	}
	defer resp.Body.Close()

	sourceRPM, sRPM, bRPM, err := util.CreateRpmBuildStructure(outputPath)
	if err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to create rpmbuild structure")
	}

	outputRpmPath := filepath.Join(sRPM, filepath.Base(url))
	out, err := os.Create(outputRpmPath)
	if err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to create output file: " + outputRpmPath)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to save file to: " + outputRpmPath)
	}

	// can we use out.filename?
	cmd := exec.Command("bash", "-c", "rpm2cpio "+outputRpmPath+" | cpio -idv")
	cmd.Dir = sourceRPM
	if err := cmd.Run(); err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to unpack rpm file")
	}

	// get the specfile
	rpmSpec, err := RpmFindAndParseSpec(sourceRPM)
	if err != nil {
		return RpmSpec{}, errors.New("RpmGetSrcRpm: failed to parse specfile: " + err.Error())
	}

	rpmSpec.SrpmLocation = sRPM
	rpmSpec.SourcesLocation = sourceRPM
	rpmSpec.OutLocation = outputPath
	rpmSpec.BuildLocation = bRPM

	/// move specfile to SPECS
	return rpmSpec, nil
}
