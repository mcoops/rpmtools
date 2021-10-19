package rpmtools

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	util "github.com/mcoops/rpmtools/internal"
)

// RpmSpec is a reference to metadata about a src rpm, including info like
// the specfile found, the rows in the file and most importantly a map of
// SpecTag structs so help easily reference values, for example
// RpmSpec.Tags["url"].
type RpmSpec struct {
	SpecLocation      string
	SrpmLocation      string
	SourcesLocation   string
	OutLocation       string
	BuildLocation     string
	Tags              map[string][]SpecTag
	SourcesTags       []SpecTag
	PatchTags         []SpecTag
	BuildRequiresTags []SpecTag
	RequiresTags      []SpecTag
	BundledCodes      []RpmNameVersion
}

// Represent a name of a tool/package/library + its version, useful for
// Provides: tags where a name and version is specified
type RpmNameVersion struct {
	Name    string
	Version string
}

// Represents a row/field of key name + key value within a specfile
type SpecTag struct {
	// Name of value in specfile, i.e. sourec0, url, summary
	TagName string
	// The actual value of the specfile definition
	TagValue string
}

var specfileLabelsRegex map[string]*regexp.Regexp

func init() {
	if _, err := exec.LookPath("rpmbuild"); err != nil {
		log.Fatal("rpmbuild is required in PATH")
	}

	if _, err := exec.LookPath("rpmspec"); err != nil {
		log.Fatal("rpmspec is required in PATH")
	}

	// https://github.com/bkircher/python-rpm-spec/blob/master/pyrpm/spec.py
	specfileLabelsRegex = make(map[string]*regexp.Regexp)
	specfileLabelsRegex["name"] = regexp.MustCompile(`^Name\s*:\s*(\S+)`)
	specfileLabelsRegex["version"] = regexp.MustCompile(`^Version\s*:\s*(\S+)`)
	specfileLabelsRegex["epoch"] = regexp.MustCompile(`^Epoch\s*:\s*(\S+)`)
	specfileLabelsRegex["release"] = regexp.MustCompile(`^Release\s*:\s*(\S+)`)
	specfileLabelsRegex["summary"] = regexp.MustCompile(`^Summary\s*:\s*(.+)`)
	specfileLabelsRegex["license"] = regexp.MustCompile(`^License\s*:\s*(.+)`)
	specfileLabelsRegex["url"] = regexp.MustCompile(`^URL\s*:\s*(\S+)`)
	specfileLabelsRegex["buildroot"] = regexp.MustCompile(`^BuildRoot\s*:\s*(\S+)`)
	specfileLabelsRegex["buildarch"] = regexp.MustCompile(`^BuildArch\s*:\s*(\S+)`)
	specfileLabelsRegex["buildRequires"] = regexp.MustCompile(`^BuildRequires\s*:\s*(.+)`)

	specfileLabelsRegex["sources"] = regexp.MustCompile(`^(Source\d*\s*):\s*(.+)`)
	specfileLabelsRegex["patches"] = regexp.MustCompile(`^(Patch\d*\s*):\s*(\S+)`)
	specfileLabelsRegex["requires"] = regexp.MustCompile(`^Requires\s*:\s*(.+)`)
	specfileLabelsRegex["conflicts"] = regexp.MustCompile(`^Conflicts\s*:\s*(.+)`)
	specfileLabelsRegex["obsoletes"] = regexp.MustCompile(`^Obsoletes\s*:\s*(.+)`)
	specfileLabelsRegex["provides"] = regexp.MustCompile(`^Provides\s*:\s*(.+)`)
	specfileLabelsRegex["packages"] = regexp.MustCompile(`^%package\s+(\S+)`)
}

// Some specfiles do weird things. If it tries to do weird things, attempt to
// clean some of it so that rpmspec will parse correctly.
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

// Given a directory to scan, find the first file ending with .spec

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

// Using the first specfile found, parse it's fields and return an struct
// allowing easy asccess to fields.
func rpmFindAndParseSpec(dir string) (RpmSpec, error) {
	specfile, err := rpmFindSpec(dir)
	if err != nil {
		return RpmSpec{}, err
	}

	spec, err := rpmParseSpec(specfile)
	return spec, err
}

// Given a specfile parse and return fields from the file
func rpmParseSpec(name string) (RpmSpec, error) {
	if !util.Exists(name) {
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
		log.Printf("rpmParseSpec: ignoring error %s", err.Error())
		// return RpmSpec{}, err
	}

	sc := bufio.NewScanner(bytes.NewReader(out))

	for sc.Scan() {
		for k, i := range specfileLabelsRegex {
			if match := i.FindStringSubmatch(sc.Text()); match != nil {
				if len(match) == 2 {
					rpm.Tags[k] = append(rpm.Tags[k], SpecTag{TagName: k, TagValue: match[1]})
				} else if len(match) == 3 {
					rpm.Tags[k] = append(rpm.Tags[k], SpecTag{TagName: match[1], TagValue: match[2]})
				}
			}
		}
	}
	rpm.SourcesTags = rpm.Tags["sources"]
	rpm.PatchTags = rpm.Tags["patches"]
	rpm.BuildRequiresTags = rpm.Tags["buildRequires"]
	rpm.RequiresTags = rpm.Tags["requires"]

	for _, t := range rpm.Tags["provides"] {
		rg := regexp.MustCompile(`bundled\((?P<name>[^#]*)\)( = (?P<version>[^#]*))?`)
		match := rg.FindStringSubmatch(t.TagValue)

		if len(match) >= 2 {
			version := ""
			if len(match) >= 4 {
				version = match[3]
			}
			rpm.BundledCodes = append(rpm.BundledCodes, RpmNameVersion{
				Name:    match[1],
				Version: version,
			})
		}
	}

	return rpm, nil
}

// Using a rpmspec obj return source0. Could be called Source0, or Source
func (rpm RpmSpec) GetSource0() (string, error) {
	if rpm.SourcesTags == nil {
		return "", errors.New("no sources")
	}

	for _, source := range rpm.SourcesTags {
		switch source.TagName {
		case "Source0":
			fallthrough
		case "Source":
			return source.TagValue, nil
		}
	}

	// don't find any? we can only assume the first value
	return rpm.SourcesTags[0].TagValue, nil
}

// Using an rpmspec obj (rpm.spec location) and an output location, extract
// the source rpm and apply patches
func (rpm RpmSpec) ApplyPatches() error {
	cmd := exec.Command("bash", "-c", "rpmbuild -bp --nodeps --define \"_topdir "+rpm.OutLocation+" \" "+rpm.SpecLocation)
	if err := cmd.Run(); err != nil {
		return errors.New("ApplyPatches: failed to run rpmbuild: " + err.Error())
	}

	// test if rpmbuild "passed", but didn't do anything, that's a fail if empty
	if err := util.DirEmpty(rpm.BuildLocation); err != nil {
		return errors.New("ApplyPatches: BUILD dir is empty: " + err.Error())
	}

	return nil
}

// Best effort removal of src rpm files.
func (rpm RpmSpec) Cleanup() {
	os.RemoveAll(rpm.SourcesLocation)
	os.RemoveAll(rpm.SrpmLocation)
	os.RemoveAll(rpm.BuildLocation)
	os.RemoveAll(filepath.Join(rpm.OutLocation, "BUILDROOT"))
	os.RemoveAll(filepath.Join(rpm.OutLocation, "RPMS"))
}

func RpmSpecFromFile(filePath string, outputPath string) (RpmSpec, error) {
	sourcesDir, srpmDir, buildDir, err := util.CreateRpmBuildStructure(outputPath)
	if err != nil {
		return RpmSpec{}, errors.New("failed to create rpmbuild structure")
	}

	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return RpmSpec{}, errors.New("failed to make path absolute")
	}
	cmd := exec.Command("bash", "-c", "rpm2cpio "+filePath+" | cpio -idv")
	cmd.Dir = sourcesDir
	if err := cmd.Run(); err != nil {
		return RpmSpec{}, errors.New("failed to unpack rpm file")
	}

	rpmSpec, err := rpmFindAndParseSpec(sourcesDir)
	if err != nil {
		return RpmSpec{}, errors.New("failed to parse specfile: " + err.Error())
	}

	rpmSpec.SrpmLocation = srpmDir
	rpmSpec.SourcesLocation = sourcesDir
	rpmSpec.OutLocation = outputPath
	rpmSpec.BuildLocation = buildDir

	return rpmSpec, nil
}
