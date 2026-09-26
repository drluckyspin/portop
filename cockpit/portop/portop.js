(function () {
    "use strict";

    const body = document.getElementById("rows");
    const filter = document.getElementById("filter");
    const status = document.getElementById("status");
    const empty = document.getElementById("empty");
    const tabs = [...document.querySelectorAll(".tab")];
    let rows = [];
    let selectedState = "all";
    let active = false;

    function cell(tr, value, className) {
        const td = document.createElement("td");
        if (className) td.className = className;
        td.textContent = value;
        tr.append(td);
        return td;
    }

    function render() {
        const query = filter.value.trim().toLowerCase();
        const matches = rows.filter(row =>
            (selectedState === "all" || row.state === selectedState) &&
            (!query || [row.local_port, row.local_address, row.pid, row.process,
                row.protocol, row.state, row.systemd_unit, row.container]
                .some(value => String(value || "").toLowerCase().includes(query))));
        const fragment = document.createDocumentFragment();

        for (const port of matches) {
            const tr = document.createElement("tr");
            const local = cell(tr, ":" + port.local_port, "port");
            const address = document.createElement("span");
            address.className = "address";
            address.textContent = port.local_address;
            local.append(address);
            cell(tr, port.protocol);
            const stateCell = cell(tr, "");
            const badge = document.createElement("span");
            badge.className = "state " + port.state.toLowerCase();
            badge.textContent = port.state;
            stateCell.append(badge);
            const process = cell(tr, port.process || "Unknown", port.process ? "process" : "muted");
            if (port.pid) {
                const pid = document.createElement("span");
                pid.className = "pid";
                pid.textContent = "PID " + port.pid;
                process.append(pid);
            }
            cell(tr, port.systemd_unit || port.container || "—", port.systemd_unit || port.container ? "" : "muted");
            fragment.append(tr);
        }

        body.replaceChildren(fragment);
        document.getElementById("visible-count").textContent = matches.length;
        empty.hidden = matches.length !== 0;
    }

    async function refresh() {
        if (active || document.hidden) return;
        active = true;
        try {
            const output = await cockpit.spawn(["portop", "--json", "--no-dns"], { err: "message" });
            rows = JSON.parse(output);
            const listening = rows.filter(row => row.state === "LISTEN").length;
            const connected = rows.filter(row => row.state === "ESTABLISHED").length;
            document.getElementById("total-count").textContent = rows.length;
            document.getElementById("listen-count").textContent = listening;
            document.getElementById("process-count").textContent = new Set(rows.map(row => row.pid).filter(Boolean)).size;
            document.getElementById("all-count").textContent = rows.length;
            document.getElementById("listening-count").textContent = listening;
            document.getElementById("connected-count").textContent = connected;
            status.classList.remove("error");
            status.textContent = "● Updated " + new Date().toLocaleTimeString();
            render();
        } catch (error) {
            status.classList.add("error");
            status.textContent = "● Could not scan ports: " + (error.message || error);
        } finally {
            active = false;
        }
    }

    filter.addEventListener("input", render);
    for (const tab of tabs) {
        tab.addEventListener("click", () => {
            selectedState = tab.dataset.state;
            for (const item of tabs) {
                const selected = item === tab;
                item.classList.toggle("active", selected);
                item.setAttribute("aria-pressed", selected);
            }
            render();
        });
    }
    document.addEventListener("visibilitychange", refresh);
    document.addEventListener("keydown", event => {
        if (event.key === "/" && document.activeElement !== filter) {
            event.preventDefault();
            filter.focus();
        }
    });
    refresh();
    setInterval(refresh, 2000);
}());
