# Security Policy

## Supported Versions

Tudou is currently maintained on the latest published code in this repository and the latest released container / binary artifacts.

## Reporting a Vulnerability

If you believe you have found a security issue, do not open a public issue first.

Please report it privately with:

- affected version or commit
- deployment mode: binary / Docker / source
- reproduction steps or proof of concept
- impact assessment
- any suggested mitigation

Preferred contact:

- GitHub Security Advisories private report

If GitHub private reporting is not available, contact the maintainer through the repository profile email and include `Tudou Security` in the subject.

## Scope

Security issues include, but are not limited to:

- authentication or authorization bypass
- JWT handling weaknesses
- request smuggling / SSRF / unsafe proxying
- secret leakage
- tenant or token isolation failures
- admin panel privilege escalation

## Response Expectations

Best effort process:

1. acknowledge receipt
2. confirm whether the report is valid
3. prepare a fix or mitigation
4. publish the fix and disclose appropriately

Please avoid publishing details until a fix or mitigation is available.
