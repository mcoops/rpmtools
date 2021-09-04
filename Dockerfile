FROM fedora:34

RUN dnf install -y golang rpm-build

COPY . .

RUN go build cmd/rpmtools/rpmtools.go
