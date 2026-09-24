const body = document.getElementById("rows");
const filter = document.getElementById("filter");
const status = document.getElementById("status");
const empty = document.getElementById("empty");
const totalCount = document.getElementById("total-count");
const listenCount = document.getElementById("listen-count");
const visibleCount = document.getElementById("visible-count");
let rows = [];
let refreshing = false;

function cell(row, value, className) {
    const td = document.createElement("td");
    if (className) td.className = className;
    td.textContent = value;
    row.append(td);
    return td;
}

function render() {
    const query = filter.value.trim().toLowerCase();
    const matches = rows.filter(row => !query ||
        String(row.local_port).includes(query) ||
        String(row.pid || "").includes(query) ||
        (row.process || "").toLowerCase().includes(query) ||
        (row.local_address || "").toLowerCase().includes(query));
    const fragment = document.createDocumentFragment();

    for (const port of matches) {
        const tr = document.createElement("tr");
        const local = cell(tr, ":" + port.local_port, "port");
        const address = document.createElement("span");
        address.className = "address";
        address.textContent = port.local_address;
        local.append(address);
        cell(tr, port.protocol);
        const state = cell(tr, "", "");
        const badge = document.createElement("span");
        badge.className = "state " + port.state.toLowerCase();
        badge.textContent = port.state;
        state.append(badge);
        cell(tr, port.process || "Unknown", port.process ? "process" : "muted");
        cell(tr, port.pid || "—", port.pid ? "" : "muted");
        cell(tr, port.cpu_percent.toFixed(1) + "%");
        cell(tr, port.systemd_unit || port.container || "—", port.systemd_unit || port.container ? "" : "muted");
        fragment.append(tr);
    }

    body.replaceChildren(fragment);
    visibleCount.textContent = matches.length;
    empty.hidden = matches.length !== 0;
}

async function refresh() {
    if (refreshing || document.hidden) return;
    refreshing = true;
    try {
        const response = await fetch("/api/ports", { cache: "no-store" });
        if (!response.ok) throw new Error("scan failed");
        rows = await response.json();
        totalCount.textContent = rows.length;
        listenCount.textContent = rows.filter(row => row.state === "LISTEN").length;
        status.classList.remove("error");
        status.textContent = "● Updated " + new Date().toLocaleTimeString();
        render();
    } catch (error) {
        status.classList.add("error");
        status.textContent = "● Could not refresh ports";
    } finally {
        refreshing = false;
    }
}

filter.addEventListener("input", render);
document.addEventListener("visibilitychange", refresh);
document.addEventListener("keydown", event => {
    if (event.key === "/" && document.activeElement !== filter) {
        event.preventDefault();
        filter.focus();
    }
});
refresh();
setInterval(refresh, 2000);
