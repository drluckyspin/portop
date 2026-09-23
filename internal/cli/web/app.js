const body = document.getElementById("rows");
const status = document.getElementById("status");
const filter = document.getElementById("filter");
let rows = [];
let refreshing = false;

function render() {
    const query = filter.value.toLowerCase();
    body.replaceChildren();
    let shown = 0;

    for (const row of rows) {
        if (query && !String(row.local_port).includes(query) &&
            !String(row.pid || "").includes(query) &&
            !(row.process || "").toLowerCase().includes(query)) {
            continue;
        }

        shown++;
        const tr = document.createElement("tr");
        const values = [
            row.protocol,
            row.local_address + ":" + row.local_port,
            row.state,
            row.pid || "—",
            row.process || "—",
            row.cpu_percent.toFixed(1) + "%",
            row.systemd_unit || row.container || "—"
        ];
        for (const value of values) {
            const td = document.createElement("td");
            td.textContent = value;
            tr.append(td);
        }
        body.append(tr);
    }
    status.textContent = shown + " sockets visible";
}

async function refresh() {
    if (refreshing || document.hidden) return;
    refreshing = true;
    try {
        const response = await fetch("/api/ports", { cache: "no-store" });
        if (!response.ok) throw new Error("scan failed");
        rows = await response.json();
        render();
    } catch (error) {
        status.textContent = "Unable to refresh ports: " + error.message;
    } finally {
        refreshing = false;
    }
}

filter.addEventListener("input", render);
document.addEventListener("visibilitychange", refresh);
refresh();
setInterval(refresh, 2000);
