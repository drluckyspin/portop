# Cockpit add-on

Build the current checkout and install the binary and page on the Linux host:

```bash
go build -o ./bin/portop ./cmd/portop
sudo install -m 755 ./bin/portop /usr/local/bin/portop
install -d "$HOME/.local/share/cockpit/portop"
cp cockpit/portop/{manifest.json,index.html,portop.css,portop.js,logo.png} "$HOME/.local/share/cockpit/portop/"
```

Run `cockpit-bridge --packages` and look for `portop`. Open
<https://localhost:9090/> on the host, or `https://<host>:9090/` from another
computer. Sign in as the same user, then select **Tools → Ports (portop)**.
Hard-refresh the browser if the page was already open.

The page calls `portop --json --no-dns` through Cockpit's bridge every two
seconds. **Details** calls `portop --inspect-pid` for command line, user,
executable, memory and other process information. Terminate and Force kill
send SIGTERM and SIGKILL through `portop --signal-pid`, after confirmation and
a check that the process start time has not changed. These actions use your
Cockpit user's permissions. A row with no PID cannot be signaled. The
Light/Dark button remembers your choice in this browser.

For a system-wide installation, copy `cockpit/portop` to
`/usr/share/cockpit/portop`. If the bridge cannot find the `portop` binary,
install it in a directory on its PATH, such as `/usr/local/bin`.
