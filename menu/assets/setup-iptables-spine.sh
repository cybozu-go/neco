#!/bin/sh

# eth9 -> BMC
iptables -t nat -A POSTROUTING -o eth9 -d {{.Bmc}} -j MASQUERADE
