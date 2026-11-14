// Pratinjau foto lampiran: klik foto -> muncul di atas halaman, ditutup dengan tombol X, Esc, atau klik area gelap.
// Tombol putar (rotate) -> putar 90° clockwise, berguna untuk foto vertikal (potrait) yang terlihat kecil.
// Scroll/rodi mouse -> zoom in/out. Klik foto -> toggle zoom 100% / fit layar.
// Saat zoom > 1x: drag/geser untuk pindah area (baca KTP, dll).
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
      <button type="button" class="lightbox-btn lightbox-btn-text" data-act="rotate" aria-label="Putar 90°">rotate</button>
      <button type="button" class="lightbox-btn" data-act="close" aria-label="Tutup">${icon('<path d="M18 6L6 18M6 6l12 12"/>')}</button>
    </div>
    <div class="lightbox-stage"><img alt=""></div>
    <button type="button" class="lightbox-nav lightbox-prev" data-act="prev" aria-label="Foto sebelumnya">${icon('<path d="M15 18l-6-6 6-6"/>')}</button>
    <button type="button" class="lightbox-nav lightbox-next" data-act="next" aria-label="Foto berikutnya">${icon('<path d="M9 6l6 6-6 6"/>')}</button>`;
  dialog.classList.toggle("single", links.length < 2);
  document.body.append(dialog);

  const img = dialog.querySelector("img");
  const caption = dialog.querySelector(".lightbox-caption");
  const stage = dialog.querySelector(".lightbox-stage");
  let index = 0;
  let scale = 1;
  let translateX = 0;
  let translateY = 0;
  let isDragging = false;
  let hasDragged = false;
  let dragStartX = 0;
  let dragStartY = 0;
  let dragTranslateX = 0;
  let dragTranslateY = 0;
  let naturalWidth = 0;
  let naturalHeight = 0;

  function show(i) {
    index = (i + links.length) % links.length;
    const link = links[index];
    img.classList.remove("zoomed", "rotated");
    img.style.transform = "";
    img.style.width = "";
    img.style.height = "";
    scale = 1;
    translateX = 0;
    translateY = 0;
    dragTranslateX = 0;
    dragTranslateY = 0;
    naturalWidth = 0;
    naturalHeight = 0;
    img.src = link.href;
    img.alt = link.querySelector("img")?.alt ?? "";
    caption.textContent = `${link.dataset.caption ?? ""} (${index + 1}/${links.length})`;
  }

  // update layout size so scroll container matches visual size (esp. when rotated)
  function updateLayoutSize() {
    if (!naturalWidth || !naturalHeight) return;
    const isRotated = img.classList.contains("rotated");
    const stageRect = stage.getBoundingClientRect();
    const maxW = stageRect.width;
    const maxH = stageRect.height;

    // effective dimensions after rotation
    const effW = isRotated ? naturalHeight : naturalWidth;
    const effH = isRotated ? naturalWidth : naturalHeight;

    // fit into stage (object-fit: contain behavior)
    const fitScale = Math.min(maxW / effW, maxH / effH);
    const layoutW = effW * fitScale;
    const layoutH = effH * fitScale;

    img.style.width = layoutW + "px";
    img.style.height = layoutH + "px";
  }

  function applyTransform() {
    const rotate = img.classList.contains("rotated") ? "rotate(90deg)" : "";
    const zoom = scale !== 1 ? `scale(${scale})` : "";
    const pan = (translateX !== 0 || translateY !== 0) ? `translate(${translateX}px, ${translateY}px)` : "";
    img.style.transform = [pan, rotate, zoom].filter(Boolean).join(" ");
  }

  // wait for image load to get natural dimensions
  img.addEventListener("load", () => {
    naturalWidth = img.naturalWidth;
    naturalHeight = img.naturalHeight;
    updateLayoutSize();
  });

  links.forEach((link, i) =>
    link.addEventListener("click", (e) => {
      if (e.ctrlKey || e.metaKey || e.shiftKey) return;
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
    else if (act === "rotate") {
      img.classList.toggle("rotated");
      scale = 1;
      translateX = 0;
      translateY = 0;
      dragTranslateX = 0;
      dragTranslateY = 0;
      img.classList.remove("zoomed");
      updateLayoutSize();
      applyTransform();
      return;
    }
    else if (e.target === dialog || e.target.classList.contains("lightbox-stage")) dialog.close();
  });

  // klik foto = toggle zoom 100% (ukuran asli) / fit layar
  img.addEventListener("click", () => {
    if (hasDragged) {
      hasDragged = false;
      return;
    }
    if (img.classList.contains("zoomed")) {
      img.classList.remove("zoomed");
      scale = 1;
      translateX = 0;
      translateY = 0;
      dragTranslateX = 0;
      dragTranslateY = 0;
    } else {
      img.classList.add("zoomed");
      scale = 1;
    }
    applyTransform();
  });

  // scroll wheel = zoom in/out (smooth) - zoom ke posisi kursor
  img.addEventListener("wheel", (e) => {
    e.preventDefault();
    if (scale > 1 || e.deltaY < 0) {
      const rect = img.getBoundingClientRect();
      const mouseX = e.clientX - rect.left - rect.width / 2;
      const mouseY = e.clientY - rect.top - rect.height / 2;
      
      const prevScale = scale;
      if (e.deltaY < 0) {
        scale = Math.min(scale * 1.1, 5);
      } else {
        scale = Math.max(scale / 1.1, 1);
      }
      
      if (scale > 1) {
        const factor = scale / prevScale;
        translateX = translateX * factor - mouseX * (factor - 1);
        translateY = translateY * factor - mouseY * (factor - 1);
        img.classList.add("zoomed");
      } else {
        scale = 1;
        translateX = 0;
        translateY = 0;
        dragTranslateX = 0;
        dragTranslateY = 0;
        img.classList.remove("zoomed");
      }
      applyTransform();
    }
  }, { passive: false });

  // drag/pan saat zoom > 1
  img.addEventListener("mousedown", (e) => {
    if (scale <= 1) return;
    if (e.button !== 0) return;
    isDragging = true;
    hasDragged = false;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    dragTranslateX = translateX;
    dragTranslateY = translateY;
    img.style.cursor = "grabbing";
    e.preventDefault();
  });

  window.addEventListener("mousemove", (e) => {
    if (!isDragging) return;
    const dx = e.clientX - dragStartX;
    const dy = e.clientY - dragStartY;
    if (Math.abs(dx) > 2 || Math.abs(dy) > 2) {
      hasDragged = true;
    }
    translateX = dragTranslateX + dx;
    translateY = dragTranslateY + dy;
    applyTransform();
  });

  window.addEventListener("mouseup", () => {
    if (isDragging) {
      isDragging = false;
      img.style.cursor = "grab";
    }
  });

  // touch support untuk mobile
  img.addEventListener("touchstart", (e) => {
    if (scale <= 1) return;
    if (e.touches.length !== 1) return;
    isDragging = true;
    hasDragged = false;
    dragStartX = e.touches[0].clientX;
    dragStartY = e.touches[0].clientY;
    dragTranslateX = translateX;
    dragTranslateY = translateY;
  }, { passive: false });

  window.addEventListener("touchmove", (e) => {
    if (!isDragging) return;
    if (e.touches.length !== 1) return;
    const dx = e.touches[0].clientX - dragStartX;
    const dy = e.touches[0].clientY - dragStartY;
    if (Math.abs(dx) > 2 || Math.abs(dy) > 2) {
      hasDragged = true;
    }
    translateX = dragTranslateX + dx;
    translateY = dragTranslateY + dy;
    applyTransform();
    e.preventDefault();
  }, { passive: false });

  window.addEventListener("touchend", () => {
    isDragging = false;
  });

  dialog.addEventListener("keydown", (e) => {
    if (e.key === "ArrowLeft") show(index - 1);
    if (e.key === "ArrowRight") show(index + 1);
    if (e.key === "r" || e.key === "R") {
      e.preventDefault();
      img.classList.toggle("rotated");
      scale = 1;
      translateX = 0;
      translateY = 0;
      dragTranslateX = 0;
      dragTranslateY = 0;
      img.classList.remove("zoomed");
      updateLayoutSize();
      applyTransform();
    }
    if (e.key === "+" || e.key === "=") {
      e.preventDefault();
      scale = Math.min(scale * 1.2, 5);
      img.classList.add("zoomed");
      applyTransform();
    }
    if (e.key === "-" || e.key === "_") {
      e.preventDefault();
      scale = Math.max(scale / 1.2, 1);
      if (scale <= 1) {
        scale = 1;
        translateX = 0;
        translateY = 0;
        dragTranslateX = 0;
        dragTranslateY = 0;
        img.classList.remove("zoomed");
      }
      applyTransform();
    }
    if (e.key === "0") {
      e.preventDefault();
      scale = 1;
      translateX = 0;
      translateY = 0;
      dragTranslateX = 0;
      dragTranslateY = 0;
      img.classList.remove("zoomed");
      applyTransform();
    }
    if (scale > 1) {
      const step = 50 / scale;
      if (e.key === "ArrowUp") { e.preventDefault(); translateY += step; applyTransform(); }
      if (e.key === "ArrowDown") { e.preventDefault(); translateY -= step; applyTransform(); }
    }
  });

  // update layout size on window resize (orientation change, etc)
  window.addEventListener("resize", () => {
    if (dialog.open) updateLayoutSize();
  });

  dialog.addEventListener("close", () => {
    img.removeAttribute("src");
    img.style.transform = "";
    img.style.width = "";
    img.style.height = "";
  });
  
  // set cursor style saat zoom
  const style = document.createElement("style");
  style.textContent = `
    .lightbox-stage img.zoomed { cursor: grab; }
    .lightbox-stage img.zoomed:active { cursor: grabbing; }
  `;
  document.head.appendChild(style);
})();
