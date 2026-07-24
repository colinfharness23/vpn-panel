# NOVA Ubuntu deployment

The supported production path is `deploy/ubuntu/install.sh` on Ubuntu 22.04 or
24.04 (amd64 or arm64). It installs a verified stable GitHub Release with
PostgreSQL, Nginx, Xray, automatic HTTPS, randomized internal ports and a
randomized administrator path.

Run as root:

```bash
curl -fsSL https://raw.githubusercontent.com/colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev/main/deploy/ubuntu/install.sh | env NOVA_GITHUB_REPO=colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev bash
```

Operational helpers installed by the script:

- `nova-update`
- `nova-backup`
- `nova-rollback`
- `nova-rotate-admin-path`
- `nova-finalize-domain`
- `nova-uninstall`

The installer does not modify UFW or cloud security groups. Allow TCP 80 and
443 for the site and ACME challenge, plus TCP and UDP 20000-59999 for managed lines.

Legacy upstream Docker, cloud-init, root-level installers and prebuilt Windows
utilities are intentionally not distributed by NOVA.
