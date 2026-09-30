(() => {
    "use strict";

    const themeButton = document.getElementById("theme-toggle");
    const themeKey = "portop-theme";
    let theme = "dark";
    try {
        if (localStorage.getItem(themeKey) === "light") theme = "light";
    } catch (_) {
        // The switch still works if browser storage is unavailable.
    }
    function setTheme(value) {
        theme = value;
        document.documentElement.dataset.theme = value;
        themeButton.textContent = value === "dark" ? "☀ Light" : "☾ Dark";
        themeButton.setAttribute("aria-label", `Switch to ${value === "dark" ? "light" : "dark"} theme`);
        document.querySelector('meta[name="theme-color"]').content = value === "dark" ? "#1a1b26" : "#e9eaf0";
        try { localStorage.setItem(themeKey, value); } catch (_) { /* Storage is optional. */ }
    }
    setTheme(theme);
    themeButton.addEventListener("click", () => setTheme(theme === "dark" ? "light" : "dark"));

    const password = document.getElementById("password");
    const reveal = document.getElementById("reveal");
    reveal.addEventListener("click", () => {
        const shown = password.type === "password";
        password.type = shown ? "text" : "password";
        reveal.textContent = shown ? "Hide" : "Show";
        reveal.setAttribute("aria-pressed", shown);
        password.focus();
    });

    // Resubmitting the form while the first request is in flight would count
    // as a second failed attempt against the lockout.
    const form = document.querySelector(".login-card");
    form.addEventListener("submit", () => {
        const submit = form.querySelector(".login-submit");
        submit.disabled = true;
        submit.textContent = "Signing in…";
    });
})();
