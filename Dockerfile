FROM fedora:34

RUN dnf install -y golang rpm-build

COPY . .

ENTRYPOINT ["go", "run", "cmd/rpmtools/rpmtools.go"]