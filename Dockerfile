FROM fedora:34

RUN dnf install -y golang rpm rpm-build rpm-sign

COPY . .

ENTRYPOINT ["go", "run", "cmd/rpmtools/rpmtools.go"]