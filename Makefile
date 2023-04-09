VERSION=$$(git describe --tags --always | tr -d 'v')

build:
	go build -o ./bin/nagent ./cmd/nagent
	go build -o ./bin/nagentd ./cmd/nagentd

pkg:
	cp ./bin/nagent dpkg/client/usr/bin/nagent
	cp ./bin/nagentd dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian pkg
	cp -Rvf lib/* dpkg/server/usr/share/nagentd
	IAN_DIR=dpkg/server ian pkg

release:
	cp -Rvf lib/* dpkg/server/usr/share/nagentd
	IAN_DIR=dpkg/client ian set -v ${VERSION}
	IAN_DIR=dpkg/server ian set -v ${VERSION}

	GOARM=7 go build -o bin/nagentd.armhf ./cmd/nagentd
	GOARM=7 go build -o bin/nagent.armhf ./cmd/nagent
	cp ./bin/nagent.armhf dpkg/client/usr/bin/nagent
	cp ./bin/nagentd.armhf dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian set -a armhf
	IAN_DIR=dpkg/server ian set -a armhf
	IAN_DIR=dpkg/client ian pkg
	IAN_DIR=dpkg/server ian pkg

	GOARCH=386 go build -o bin/nagentd.i386 ./cmd/nagentd
	GOARCH=386 go build -o bin/nagent.i386 ./cmd/nagent
	cp ./bin/nagent.i386 dpkg/client/usr/bin/nagent
	cp ./bin/nagentd.i386 dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian set -a i386
	IAN_DIR=dpkg/server ian set -a i386
	IAN_DIR=dpkg/client ian pkg
	IAN_DIR=dpkg/server ian pkg

	GOARCH=amd64 go build -o bin/nagentd.amd64 ./cmd/nagentd
	GOARCH=amd64 go build -o bin/nagent.amd64 ./cmd/nagent
	cp ./bin/nagent.amd64 dpkg/client/usr/bin/nagent
	cp ./bin/nagentd.amd64 dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian set -a amd64
	IAN_DIR=dpkg/server ian set -a amd64
	IAN_DIR=dpkg/client ian pkg
	IAN_DIR=dpkg/server ian pkg

	GOARCH=arm64 go build -o bin/nagentd.aarch64 ./cmd/nagentd
	GOARCH=arm64 go build -o bin/nagent.aarch64 ./cmd/nagent
	cp ./bin/nagent.aarch64 dpkg/client/usr/bin/nagent
	cp ./bin/nagentd.aarch64 dpkg/server/usr/bin/nagentd
	IAN_DIR=dpkg/client ian set -a aarch64
	IAN_DIR=dpkg/server ian set -a aarch64
	IAN_DIR=dpkg/client ian pkg
	IAN_DIR=dpkg/server ian pkg

	mkdir release
	rm -f release/*
	cp bin/nagent.armhf release/nagent.${VERSION}.armhf
	cp bin/nagentd.armhf release/nagentd.${VERSION}.armhf
	cp bin/nagent.i386 release/nagent.${VERSION}.i386
	cp bin/nagentd.i386 release/nagentd.${VERSION}.i386
	cp bin/nagent.amd64 release/nagent.${VERSION}.amd64
	cp bin/nagentd.amd64 release/nagentd.${VERSION}.amd64
	cp bin/nagent.aarch64 release/nagent.${VERSION}.aarch64
	cp bin/nagentd.aarch64 release/nagentd.${VERSION}.aarch64
	cp dpkg/server/pkg/nagentd_${VERSION}_armhf.deb release
	cp dpkg/client/pkg/nagent_${VERSION}_armhf.deb release
	cp dpkg/server/pkg/nagentd_${VERSION}_i386.deb release
	cp dpkg/client/pkg/nagent_${VERSION}_i386.deb release
	cp dpkg/server/pkg/nagentd_${VERSION}_amd64.deb release
	cp dpkg/client/pkg/nagent_${VERSION}_amd64.deb release
	cp dpkg/server/pkg/nagentd_${VERSION}_aarch64.deb release
	cp dpkg/client/pkg/nagent_${VERSION}_aarch64.deb release