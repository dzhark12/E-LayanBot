// Pratinjau foto lampiran: klik foto -> muncul di atas halaman, ditutup dengan tombol X, Esc, atau klik area gelap.
(() => {
  const links = [...document.querySelectorAll("a[data-lightbox]")];
  if (!links.length || typeof HTMLDialogElement === "undefined") return; // tanpa JS/dialog, link tetap membuka foto di tab baru

  const icon = (path) =>
    `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${path}</svg>`;

  const dialog = document.createElement("dialog");
  dialog.className = "lightbox";
  dialog.setAttribute("aria-label", "Pratinjau foto");
  dialog.innerHTML = `
    <div class="lightbox-bar">
      <span class="lightbox-caption"></span>
      <button type="button" class="lightbox-btn" data-act="close" aria-label="Tutup">${icon('<path d="M6 6l12 12M18 6L6 18"/>')}</button>
    </div>
    <div class="lightbox-stage"><img alt=""></div>
    <button type="button" class="lightbox-nav lightbox-prev" data-act="prev" aria-label="Foto sebelumnya">${icon('<path d="M15 5l-7 7 7 7"/>')}</button>
    <button type="button" class="lightbox-nav lightbox-next" data-act="next" aria-label="Foto berikutnya">${icon('<path d="M9 5l7 7-7 7"/>')}</button>`;
  dialog.classList.toggle("single", links.length < 2);
  document.body.append(dialog);

  const img = dialog.querySelector("img");
  const caption = dialog.querySelector(".lightbox-caption");
  let index = 0;

  function show(i) {
    index = (i + links.length) % links.length;
    const link = links[index];
    img.classList.remove("zoomed");
    img.src = link.href;
    img.alt = link.querySelector("img")?.alt ?? "";
    caption.textContent = `${link.dataset.caption ?? ""} (${index + 1}/${links.length})`;
  }

  links.forEach((link, i) =>
    link.addEventListener("click", (e) => {
      if (e.ctrlKey || e.metaKey || e.shiftKey) return; // tetap bisa dibuka di tab baru kalau memang mau
      e.preventDefault();
      show(i);
      dialog.showModal();
    })
  );

  dialog.addEventListener("click", (e) => {
    const act = e.target.closest("[data-act]")?.dataset.act;
    if (act === "close") dialog.close();
    else if (act === "prev") show(index - 1);
    else if (act === "next") show(index + 1);
    else if (e.target === dialog || e.target.classList.contains("lightbox-stage")) dialog.close();
  });

  // klik foto = perbesar ke ukuran asli (berguna untuk membaca tulisan kecil di KTP), klik lagi = muat layar
  img.addEventListener("click", () => img.classList.toggle("zoomed"));

  dialog.addEventListener("keydown", (e) => {
    if (e.key === "ArrowLeft") show(index - 1);
    if (e.key === "ArrowRight") show(index + 1);
  });

  dialog.addEventListener("close", () => img.removeAttribute("src")); // Esc juga lewat sini
})();
