# rpmtools

Provides basic functionality to download and extract a source RPM file, whilst
applying any patch files found.

## Dependencies

The tool `rpmbuild` must be installed. I didn't want to recreate what `rpmbuild`
or `rpmspec` does, especially parsing specfiles, and `python-rpm-spec` didn't
seem to correctly parse all marcos. 

## Library

Given a URL to a src rpm, downloads to the output location directly into the 
rpmbuild build structure i.e. /tmp/SRPM. Also parses the specfile and makes
those fields available in an `RpmSpec` struct
```go
RpmGetSrcRpm("http://ftp.iinet.net.au/pub/fedora/linux/updates/34/Modular/SRPMS/Packages/c/cri-o-1.20.0-1.module_f34+10489+4277ba4d.src.rpm", "/tmp/")
```

Attempt to work out which field in the specfile is `source0`:

```go
source0, _ := r.RpmGetSource0()
```

Get license information: 

```go
r.Tags["license"]
```

Apply any patches associated in the source rpm:

```go
if err := r.RpmApplyPatches(); err != nil {
    fmt.Println(err.Error())
}
```

Cleanup any rpmbuild build folders, i.e. reset state before `RpmGetSrcRpm`

```go
r.RpmCleanup()
```

## Dev container

Due to the reliance on rpm, it's easier just to use the dev container
functionality of VSCode to do dev. Or just develop on an RPM system.