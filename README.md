# p2p-forwarder

[![Go Report Card](https://goreportcard.com/badge/github.com/nickname32/discordhook)](https://goreportcard.com/report/github.com/nickname32/p2p-forwarder)

Tool for farwarding ports. Made using [libp2p](https://github.com/libp2p/go-libp2p).

![screenshot](https://i.imgur.com/BMFXKNK.png)

## How it works

- A: opens desired ports ports inside P2P Forwarder
- A: shares it's id from P2P Forwarder with B
- B: connects to A's id inside P2P Forwarder
- B: connect to opened ports on A's machine using address like 127.0.89.N:PORT_ON_A's_MACHINE

P.S. every edit field handles Ctrl+C and Ctrl+V. To exit the program, press Ctrl+Q

## Project status

Project is on early beta stage. I recommend you to use it only for personal purposes like playing Minecraft with friends.

Current ui is using [clui](https://github.com/VladimirMarkelov/clui), but i'm working on more comfortable one, which will be using [gotk3](https://github.com/gotk3/gotk3) (bindings to gtk3)

Also i'm thinking about making cli.

### Feel free to contribute this project by opening an issue (it can be also a question) or creating pull requests

#### Star this project if you liked it or found it useful for you :3
