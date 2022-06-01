
build:
	go build -o dpkg/client/usr/bin/nagent ./cmd/nagent
	go build -o dpkg/server/usr/bin/nagentd ./cmd/nagentd

pkg:
	IAN_DIR=dpkg/client ian pkg
	cp -Rvf lib/* dpkg/server/usr/share/nagentd
	IAN_DIR=dpkg/server ian pkg