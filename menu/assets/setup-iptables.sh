#!/bin/sh

# eth0 -> internet
# eth1 -> BMC
iptables -t nat -A POSTROUTING -o eth0 -s {{.Internet}} -j MASQUERADE
# This rule SNAT packets destined to the internet from NTP servers so that those packets can reach the internet.
iptables -t nat -A POSTROUTING -o eth0 -s {{.NTP}} -j MASQUERADE
iptables -t nat -A POSTROUTING -o eth1 -j MASQUERADE
