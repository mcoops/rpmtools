package rpmtools

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type test struct {
	url        string
	specName   string
	sourceTags []string
	patchTags  []string
}

var outDir string

var tests = []test{
	{
		"https://kojipkgs.fedoraproject.org//packages/python-urllib3/1.25.8/5.fc33/src/python-urllib3-1.25.8-5.fc33.src.rpm",
		"python-urllib3.spec",
		[]string{"https://github.com/urllib3/urllib3/archive/1.25.8/urllib3-1.25.8.tar.gz", "ssl_match_hostname_py3.py"},
		[]string{"CVE-2021-33503.patch"},
	},
	{
		"https://kojipkgs.fedoraproject.org//packages/rizin/0.2.0/2.fc33/src/rizin-0.2.0-2.fc33.src.rpm",
		"rizin.spec",
		[]string{"https://github.com/rizinorg/rizin/releases/download/v0.2.0/rizin-src-v0.2.0.tar.xz"},
		[]string{"rizin-avoid-symbols-clashing.patch"},
	},
}

func SortCompare(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func equalSpecTags(a, b []SpecTag) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestMain(m *testing.M) {
	var err error
	outDir, err = ioutil.TempDir("", "")
	if err != nil {
		return
	}
	m.Run()
	os.Remove(outDir)
}

func TestRpmSpecFromFile(t *testing.T) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("url=%s", test.url), func(t *testing.T) {
			url := test.url
			resp, err := http.Get(url)
			if err != nil {
				t.Error(err)
			}
			defer resp.Body.Close()

			out, err := ioutil.TempFile("", "")
			if err != nil {
				t.Error(err)
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				t.Error(err)
			}

			rpmSpec, err := RpmSpecFromFile(out.Name(), outDir)
			if err != nil {
				t.Error(err)
			}

			sourcesLocationExp := filepath.Join(outDir, "SOURCES")
			srpmLocationExp := filepath.Join(outDir, "SRPMS")
			buildLocationExp := filepath.Join(outDir, "BUILD")
			outLocationExp := outDir
			if rpmSpec.SourcesLocation != sourcesLocationExp {
				t.Errorf("SourcesLocation is wrong: %s, exp: %s", rpmSpec.SourcesLocation, sourcesLocationExp)
			}
			if rpmSpec.SrpmLocation != srpmLocationExp {
				t.Errorf("SRPMLocation is wrong: %s, exp: %s", rpmSpec.SrpmLocation, srpmLocationExp)
			}
			if rpmSpec.BuildLocation != buildLocationExp {
				t.Errorf("BuildLocation is wrong: %s, exp: %s", rpmSpec.SrpmLocation, buildLocationExp)
			}
			if rpmSpec.OutLocation != outLocationExp {
				t.Errorf("OutLocation is wrong: %s, exp: %s", rpmSpec.SrpmLocation, buildLocationExp)
			}
			if filepath.Base(rpmSpec.SpecLocation) != test.specName {
				t.Errorf("Spec file was wrongly detected: %s, exp: %s", filepath.Base(rpmSpec.SpecLocation), test.specName)
			}

			source0, err := rpmSpec.GetSource0()
			if err != nil {
				t.Fatal(err)
			}
			if source0 != test.sourceTags[0] {
				t.Errorf("Source0 was not correctly found: %s, exp: %s", source0, test.sourceTags[0])
			}

			if !equalSpecTags(rpmSpec.Tags["sources"], rpmSpec.SourcesTags) {
				t.Errorf("tags['sources'] different from SourcesTags")
			}
			if !equalSpecTags(rpmSpec.Tags["patches"], rpmSpec.PatchTags) {
				t.Errorf("tags['patches'] different from PatchTags")
			}
			if !equalSpecTags(rpmSpec.Tags["requires"], rpmSpec.RequiresTags) {
				t.Errorf("tags['requires'] different from RequiresTags")
			}

			sourcesTags := make([]string, len(rpmSpec.SourcesTags))
			for i, s := range rpmSpec.SourcesTags {
				sourcesTags[i] = s.TagValue
			}
			if !SortCompare(sourcesTags, test.sourceTags) {
				t.Errorf("Sources tags are wrong: %v, exp: %v", sourcesTags, test.sourceTags)
			}

			patchesTags := make([]string, len(rpmSpec.PatchTags))
			for i, s := range rpmSpec.PatchTags {
				patchesTags[i] = s.TagValue
			}
			if !SortCompare(patchesTags, test.patchTags) {
				t.Errorf("Patch tags are wrong: %v, exp: %v", patchesTags, test.patchTags)
			}

			rpmSpec.Cleanup()
		})
	}
}
