# Cockpit add-on

Install `portop` on the Linux host, then copy the `portop` directory here to
`/usr/share/cockpit/portop` (system-wide) or
`~/.local/share/cockpit/portop` (for one user). Reload Cockpit and open
**Tools → Ports (portop)**.

The page calls `portop --json --no-dns` through Cockpit's bridge every two
seconds. It uses the current Cockpit user's permissions to inspect sockets.
It is read-only. If `portop` is outside the bridge's PATH, install or symlink
it into a standard binary directory such as `/usr/local/bin`.
