# Security Policy

## Supported versions

| Version line | Supported |
| --- | --- |
| `1.x` | Yes |
| `< 1.0.0` | No |

## Reporting a vulnerability

Do not open a public GitHub issue for a security-sensitive report.

Instead:

1. collect the affected version, environment, reproduction steps, and impact
2. include logs or screenshots only if they do not expose secrets
3. send the report privately to the maintainer contact attached to the repository

If no private contact is configured yet, open a minimal issue that only says a private security contact path is needed, without disclosing the vulnerability details.

## What to expect

- initial acknowledgement target: within 3 business days
- follow-up once severity and reproduction are confirmed
- coordinated fix and disclosure timing when applicable

## Scope notes

Reports are especially valuable when they involve:

- command execution flow
- unsafe file write paths
- prompt or context leakage
- credential handling
- release pipeline or artifact integrity
