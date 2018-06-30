# BinkyNet Network Manager

The BinkyNet Network Manager (from now on called network manager) is a service
used to configure Binky Local Workers (local workers).

When a local worker starts, it sends out a network broadcast
to discover the network manager.

In response, the network manager calls an HTTP server on the local worker
that is listening on a port provided in the initial broadcast.
In this HTTP call, the network manager provides the configuration
for the local worker and the address of an MQTT server to subscribe to.

The configuration of a local worker consists of one or more `devices`
and one or more `objects` that use these `devices`.

A `device` in an actual piece of hardware that is connected to a bus (typically I2C)
controlled by the local worker.

An `object` is real world entity that uses one or more inputs/outputs of
one or more `devices` to be controlled.
Examples of `objects` are:

- A LED
- A servo
- A switch with relays controlling the electrical connections
- A sensor

## Usage

```bash
./bnManager --mqtt-host=mqtt.local --endpoint=http://$IP:8823
```
