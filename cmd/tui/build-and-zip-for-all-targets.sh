#!/bin/sh

rm -r ./builds

build() {
    if [ "$1" = "windows" ]; then
        ./build.sh "$1" "$2" "./builds/P2P Forwarder.exe"
        cd ./builds
        zip "./P2P_Forwarder_$1_$2.zip" "./P2P Forwarder.exe"
        cd ..
    else
        ./build.sh "$1" "$2" "./builds/P2P Forwarder"
        cd ./builds
        zip "./P2P_Forwarder_$1_$2.zip" "./P2P Forwarder"
        cd ..
    fi
}

build windows 386
build windows amd64

build darwin 386
build darwin amd64

build linux 386
build linux amd64
build linux arm
build linux arm64

rm "./builds/P2P Forwarder.exe"
rm "./builds/P2P Forwarder"
