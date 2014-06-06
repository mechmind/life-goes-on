Life goes on 
============

..despite the apocalypse. This is a postapoc strategy game prototype. For now it runs some zed
outbreak

Running
=======

1. Install [go](http://golang.org)
2. Set GOPATH (`export GOPATH=~/go`)
3. Fetch project `go get -u github.com/mechmind/life-goes-on`
4. Run it! `cd $GOPATH/bin && ./life-goes-on`

Game controls
=============

Mouse: left button to set next waypoint, right mouse to set grenade target. Soldier will throw
grenade when it can. Set gren target inside house, run around and see what happens :)

'w', 'a', 's', 'd' for moving window.

'f' will change firing mode. Default is staying and firing at foes, secondary - alternately fire and move

Multiplayer
===========

The game have built-in server. To start it, pass `-listen ADDR ` option on start. ADDR can be as 
simple as port definition, i.e. `:4242`. Clients than connect to server using option `-connect IP:PORT`.

Game rules
==========

There are various game rules that can alternate the gameplay. Admin can add rules using
`-rule RULENAME` option. Multiple rules can be set and then they will be selected in a round-robin.
Available rules can be dumped using `-dump-rules` option.
