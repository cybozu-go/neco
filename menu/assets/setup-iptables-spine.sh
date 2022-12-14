#!/bin/sh

# eth1 -> BMC
iptables -t nat -A POSTROUTING -o eth1 -d {{.}} -j MASQUERADE
