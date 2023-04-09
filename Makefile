
build:
	go build -o ./bin/nagent ./cmd/nagent
	go build -o ./bin/nagentd ./cmd/nagentd

pkg:
	cp ./bin/nagent dpkg/client/usr/bin/nagent
	cp ./bin/nagentd dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian pkg
	cp -Rvf lib/* dpkg/server/usr/share/nagentd
	IAN_DIR=dpkg/server ian pkg