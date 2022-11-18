# UDPX
![Alt text](screenshots/showcase.png)
Fast, single-packet UDP scanner written in Go. Supports discovery of more than 45 services with the possibility to add your own. It is lightweight - grab a binary and run it anywhere you want. Linux, Mac Os and Windows are supported but can be built for more platforms. The UDPX is not an internet-wide port scanner such as zgrab2 or zmap. The goal was to make easy to use and portable UDP scanner.

* It is fast. It can scan whole /16 network in ~20 seconds for a single service.
* You don't need to instal libpcap or any other dependencies.
* Can run on Linux, Mac Os, Windows. Or your Nethunter if you built it for Arm.
* Customizable. You can add your probes and test for even more protocols!

## How it works
Scanning UDP ports is very different than scanning TCP - you may, or may not get any result back from probing an UDP port as UDP is a connectionless protocol. UDPX implements a single-packet based approach. A protocol-specific packet is sent to the defined service (port) and waits for a response. The limit is set to 500 ms by default and can be changed by `-w` flag. If the service sends a packet back within this time, it is certain that it is indeed listening on that port and is reported as open.

A typical technique is to send 0 byte UDP packets to each port on the target machine. If we receive an "ICMP Port Unreachable" message, then the port is closed. If an UDP response is received to the probe (unusual), the port is open. If we get no response at all, the state is open or filtered, meaning that the port is either open or packet filters are blocking the communication. This method is not implemented as there is no added value (UDPX tests only for specific protocols).

## Usage

To scan a single IP:
```
udpx -t 1.1.1.1
```

To scan a CIDR with maximum of 128 connections and timeout of 1000 ms:
```
udpx -t 1.2.3.4/24 -c 128 -w 1000
```

To scan targets from file with maximum of 128 connections for only specific service:
```
udpx -tf targets.txt -c 128 -s ipmi
```

Target can be:
* IP address
* CIDR
* Domain

IPv6 is supported.

## Options
```

        __  ______  ____ _  __
       / / / / __ \/ __ \ |/ /
      / / / / / / / /_/ /   /
     / /_/ / /_/ / ____/   |
     \____/_____/_/   /_/|_|
         v1.0.2-beta, by @nullt3r

Usage of ./udpx-linux-amd64:
  -c int
    	Maximum number of concurrent connections (default 32)
  -nr
    	Do not randomize addresses
  -o string
    	Output file to write results
  -s string
    	Scan only for a specific service, one of: ard, bacnet, bacnet_rpm, chargen, citrix, coap, db, db, digi1, digi2, digi3, dns, ipmi, ldap, mdns, memcache, mssql, nat_port_mapping, natpmp, netbios, netis, ntp, ntp_monlist, openvpn, pca_nq, pca_st, pcanywhere, portmap, qotd, rdp, ripv, sentinel, sip, snmp1, snmp2, snmp3, ssdp, tftp, ubiquiti, ubiquiti_discovery_v1, ubiquiti_discovery_v2, upnp, valve, wdbrpc, wsd, wsd_malformed, xdmcp, kerberos, ike
  -sp
    	Show received packets (only first 32 bytes)
  -t string
    	IP/CIDR to scan
  -tf string
    	File containing IPs/CIDRs to scan
  -w int
    	Maximum time to wait for a response (socket timeout) in ms (default 500)
```

## Building
You can grab prebuilt binaries in the release section. If you want to build UDPX from source, follow these steps:

From git:
```
git clone https://github.com/nullt3r/udpx
cd udpx
go build ./cmd/udpx
```
You can find the binary in the current directory.

Or via go:
```
go install -v github.com/nullt3r/udpx/cmd/udpx@latest
```

After that, you can find the binary in `$HOME/go/bin/udpx`. If you want, move binary to `/usr/local/bin/` so you can call it directly.

## Supported services
The UDPX supports more then 45 services. The most interesting are:
* ipmi
* snmp
* ike
* tftp
* openvpn
* kerberos
* ldap

The complete list of supported services:
* ard
* bacnet
* bacnet_rpm
* chargen
* citrix
* coap
* db
* db
* digi1
* digi2
* digi3
* dns
* ipmi
* ldap
* mdns
* memcache
* mssql
* nat_port_mapping
* natpmp
* netbios
* netis
* ntp
* ntp_monlist
* openvpn
* pca_nq
* pca_st
* pcanywhere
* portmap
* qotd
* rdp
* ripv
* sentinel
* sip
* snmp1
* snmp2
* snmp3
* ssdp
* tftp
* ubiquiti
* ubiquiti_discovery_v1
* ubiquiti_discovery_v2
* upnp
* valve
* wdbrpc
* wsd
* wsd_malformed
* xdmcp
* kerberos
* ike