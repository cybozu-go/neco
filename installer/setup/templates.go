package main

import "text/template"

var (
	birdConfTemplate = template.Must(template.New("bird.conf").Parse(`# bird configurations for boot servers
log stderr all;
protocol device {
    scan time 60;
}
protocol direct singles {
    ipv4;
    interface "node0", "bastion", "test";
}
protocol bfd {
    interface "*" {
       min rx interval 400 ms;
       min tx interval 400 ms;
    };
}
protocol kernel {
    merge paths;
    ipv4 {
        export filter {
            if source = RTS_DEVICE then reject;
            accept;
        };
    };
}

ipv4 table dummytab;
protocol static dummystatic {
    ipv4 { table dummytab; };
    route 0.0.0.0/0 via "lo";
}
template bgp tor {
    local as {{ .ASN }};
    direct;
    bfd;
    error wait time 3,300;

    ipv4 {
        # Accept routes regardless of its NEXT_HOP.
        igp table dummytab;
        gateway recursive;

        import filter {
            # If this route came from iBGP peers,
            if bgp_next_hop.mask({{ .Mask }}) = from.mask({{ .Mask }}) then {
                # use the NEXT_HOP as the gateway address.
                gw = bgp_next_hop;
                accept;
            }

            # Otherwise, use the router address as the gateway address.
            # This is virtually equal to "next hop self" on receiver side.
            gw = from;
            accept;
        };
        export all;
    };
}
protocol bgp tor1 from tor {
    neighbor {{ .ToR1 }} as {{ .ASN }};
}
protocol bgp tor2 from tor {
    neighbor {{ .ToR2 }} as {{ .ASN }};
}
`))

	chronyConfTemplate = template.Must(template.New("chrony.conf").Parse(`# Welcome to the chrony configuration file. See chrony.conf(5) for more
# information about usuable directives.
{{ range . }}
server {{ . }} iburst
{{- end }}

# This directive specify the location of the file containing ID/key pairs for
# NTP authentication.
keyfile /etc/chrony/chrony.keys

# This directive specify the file into which chronyd will store the rate
# information.
driftfile /var/lib/chrony/chrony.drift

# Uncomment the following line to turn logging on.
#log tracking measurements statistics

# Log files location.
logdir /var/log/chrony

# Stop bad estimates upsetting machine clock.
maxupdateskew 100.0

# This directive enables kernel synchronisation (every 11 minutes) of the
# real-time clock. Note that it can’t be used along with the 'rtcfile' directive.
rtcsync

# Step the system clock instead of slewing it if the adjustment is larger than
# 0.1 seconds, but only in the first three clock updates.
makestep 0.1 3

# Allow NTP client access from local network.
allow 10.0.0.0/8

# Ignore leap second; adjust by slewing
leapsecmode slew
maxslewrate 1000
smoothtime 400 0.001 leaponly

# mlockall
lock_all

# set highest scheduling priority
sched_priority 99
`))
)
