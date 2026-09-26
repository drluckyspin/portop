# Cockpit add-on

To try the version in this checkout on a Linux host with Cockpit installed:

```bash
go build -o ./bin/portop ./cmd/portop
sudo install -m 755 ./bin/portop /usr/local/bin/portop
install -d "$HOME/.local/share/cockpit/portop"
cp cockpit/portop/{manifest.json,index.html,portop.css,portop.js} "$HOME/.local/share/cockpit/portop/"
```

Check that Cockpit can find the page with `cockpit-bridge --packages` (look for
`portop`). Open <https://localhost:9090/> on the host, or
`https://<host>:9090/` from another computer, sign in as the same user, and
select **Tools → Ports (portop)**. Reload Cockpit in the browser if the page
was already open. The page should show the host's ports and refresh every two
seconds. Compare with `/usr/local/bin/portop --json --no-dns` in a terminal on
that host. Both commands run with your user's permissions.

Install `portop` on the Linux host, then copy the `portop` directory here to
`/usr/share/cockpit/portop` (system-wide) or
`~/.local/share/cockpit/portop` (for one user). Reload Cockpit and open
**Tools → Ports (portop)**.

The page calls `portop --json --no-dns` through Cockpit's bridge every two
seconds. It uses the current Cockpit user's permissions to inspect sockets.
It is read-only. If `portop` is outside the bridge's PATH, install or symlink
it into a standard binary directory such as `/usr/local/bin`.
