# rpmtools

Provides basic functionality to download and extract a source RPM file, whilst
applying any patch files found.

## Dependencies

The tool `rpmbuild` must be installed. I didn't want to recreate what `rpmbuild`
or `rpmspec` does, especially parsing specfiles, and `python-rpm-spec` didn't
seem to correctly parse all macros.

## Library

Given a path to a local file it extracts the SRPM in the output directory and
parses the specfile and makes those fields available in an `RpmSpec` struct
```go
RpmSpecFromFile("cri-o-1.20.0-1.module_f34+10489+4277ba4d.src.rpm", "/tmp/")
```

Attempt to work out which field in the specfile is `source0`:

```go
source0, _ := r.GetSource0()
```

Get license information: 

```go
r.Tags["license"]
```

Apply any patches associated in the source rpm:

```go
if err := r.ApplyPatches(); err != nil {
    fmt.Println(err.Error())
}
```

Cleanup any rpmbuild build folders, i.e. reset state before `RpmSpecFromFile`

```go
r.Cleanup()
```

## Dev container

Due to the reliance on rpm, it's easier just to use the dev container
functionality of VSCode to do dev. Or just develop on an RPM system.