# Security Policy

HydraDNS is a DNS-layer security tool, so please report vulnerabilities privately
rather than opening a public issue.

## Reporting

Email **inbox@roshansingh.systems** with a description, affected version or commit,
and reproduction steps. Please do not disclose the issue publicly until it has been
addressed.

## Scope

This project is pre-1.0 and moving fast. The DNS resolver, the gRPC control plane, and
the blocklist and policy engines are the areas where a security report is most valuable.

## Known limitation

HydraDNS filters at the DNS layer, so it cannot stop a client that hardcodes a DoH
resolver by raw IP. Pair it with a firewall rule on ports 443 and 853 to close that path.
This is documented behaviour, not a vulnerability.
