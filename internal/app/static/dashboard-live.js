const dashboard = document.querySelector("[data-request-version]");
if (dashboard) {
  const current = Number(dashboard.dataset.requestVersion);
  setInterval(async () => {
    try {
      const response = await fetch("/admin/requests/version", { cache: "no-store" });
      if (!response.ok) return;
      const { version } = await response.json();
      if (version !== current) window.location.reload();
    } catch {}
  }, 10000);
}
