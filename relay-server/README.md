# Relay Server with libp2p

This creates a relay server.

## Build

From the `perun-eth-mobile` directory run the following:

```
> cd relay-server/
> go build
```
Windows Users have to rename the file into a .exe file.

## Usage

```
> ./relay-server
------------------------
Listening on port: 5574
Relay Station is now running...
------------------------
Run: 'relay-server.exe -cli -id QmWBHAqpi4EHxKBufoHdYT2WjqapBy3QsUefdmKB4bX3Ja -addr /ip4/77.182.187.157/tcp/5574' on another console to get a client to connect to this relay server.
------------------------
```

The listener libp2p host will print its `Multiaddress`, which indicates how it can be reached (ip4+tcp) and its randomly generated ID (`QmYo41Gyb...`)

Now, launch another node that talks to the listener:

```
TODO
```

The new node with send the message ...

## Details
