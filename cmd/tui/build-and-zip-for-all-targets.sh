#!/bin/sh

rm -r ./builds

build() {
    if [ "$1" = "windows" ]; then
        ./build.sh "$1" "$2" "./builds/P2P Forwarder.exe"
        zip "./builds/P2P_Forwarder_$1_$2.zip" "./builds/P2P Forwarder.exe"
    else
        ./build.sh "$1" "$2" "./builds/P2P Forwarder"
        zip "./builds/P2P_Forwarder_$1_$2.zip" "./builds/P2P Forwarder"
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
