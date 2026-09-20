(function () {
    "use strict";
    const body = document.getElementById("rows");
    const status = document.getElementById("status");
    const filter = document.getElementById("filter");
    let rows = [];
    let active = false;

    function render() {
        const query = filter.value.toLowerCase();
        body.replaceChildren();
        let shown = 0;
        for (const row of rows) {
            if (query && !String(row.local_port).includes(query) &&
                !String(row.pid || "").includes(query) &&
                !(row.process || "").toLowerCase().includes(query)) continue;
            shown++;
            const tr = document.createElement("tr");
            for (const value of [row.protocol, row.local_address + ":" + row.local_port,
                row.state, row.pid || "—", row.process || "—", row.systemd_unit || row.container || "—"]) {
                const td = document.createElement("td");
                td.textContent = value;
                tr.append(td);
            }
            body.append(tr);
        }
        status.textContent = shown + " sockets visible";
    }

    async function refresh() {
        if (active || document.hidden) return;
        active = true;
        try {
            const output = await cockpit.spawn(["portop", "--json", "--no-dns"], { err: "message" });
            rows = JSON.parse(output);
            render();
        } catch (error) {
            status.textContent = "Could not scan ports: " + error.message;
        } finally {
            active = false;
        }
    }

    filter.addEventListener("input", render);
    document.addEventListener("visibilitychange", refresh);
    refresh();
    setInterval(refresh, 2000);
}());
