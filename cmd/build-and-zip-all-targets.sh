#!/bin/sh

SOURCE_PATH=$(realpath "$1")
BUILDS_PATH=$(realpath "$2")
FILENAME=$3

MAIN_PATH=$PWD

rm -r "$BUILDS_PATH"

build() {
    if [ "$1" = "windows" ]; then
        cd "$SOURCE_PATH"
        "$MAIN_PATH/build.sh" "$1" "$2" "$BUILDS_PATH/$FILENAME.exe"
        cd "$BUILDS_PATH"
        zip "./${FILENAME}_$1_$2.zip" "./$FILENAME.exe"
        cd "$MAIN_PATH"
    else
        cd "$SOURCE_PATH"
        "$MAIN_PATH/build.sh" "$1" "$2" "$BUILDS_PATH/$FILENAME"
        cd "$BUILDS_PATH"
        zip "./${FILENAME}_$1_$2.zip" "./$FILENAME"
        cd "$MAIN_PATH"
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

rm "$BUILDS_PATH/$FILENAME.exe"
rm "$BUILDS_PATH/$FILENAME"
