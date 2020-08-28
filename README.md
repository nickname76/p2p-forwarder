# P2P Forwarder

[![Go Report Card](https://goreportcard.com/badge/github.com/nickname32/discordhook)](https://goreportcard.com/report/github.com/nickname32/p2p-forwarder)

A tool for farwarding ports. Made using [libp2p](https://github.com/libp2p/go-libp2p).

![screenshot](https://repository-images.githubusercontent.com/284020308/1534a100-d34b-11ea-9d0f-b22749e919b9)

## How it works

- A: opens desired ports ports inside P2P Forwarder
- A: shares it's id from P2P Forwarder with B
- B: connects to A's id inside P2P Forwarder
- B: connect to opened ports on A's machine using address like 127.0.89.N:PORT_ON_A's_MACHINE

P.S. every edit field handles Ctrl+C and Ctrl+V. To exit the program, press Ctrl+Q

## Project status

Project is on early beta stage. I recommend you to use it only for personal purposes like playing Minecraft with friends.

Current uis are cli and tui ([clui](https://github.com/VladimirMarkelov/clui)).

### Feel free to contribute to this project by opening an issue (it can be a question) or creating a pull requests

#### Star this project if you liked it or found it useful for you :3
