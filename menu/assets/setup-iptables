#!/bin/sh

# eth0 -> internet
# eth1 -> BMC
iptables -t nat -A POSTROUTING -o eth0 -s {{.}} -j MASQUERADE
iptables -t nat -A POSTROUTING -o eth1 -j MASQUERADE
