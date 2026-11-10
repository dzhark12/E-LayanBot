/* Validasi form "Data surat" E-LAYAN.
 * Hanya menangani form /requests/:id/identity agar fitur lain tetap utuh.
 * Aturan berdasarkan nama/label field yang dibentuk oleh backend.
 */
(() => {
  "use strict";

  const form = document.querySelector('form.identity-form[action$="/identity"]');
  if (!form) return;

  const RELIGIONS = ["Islam", "Kristen", "Katolik", "Hindu", "Buddha", "Khonghucu"];
  const LETTER = /\p{L}/u;
  const NAME_CHARS = /^[\p{L}\p{M}\s.'’\-]+$/u;
  const PLACE_CHARS = /^[\p{L}\p{M}\p{N}\s.'’\-/,()]+$/u;
  const JOB_CHARS = /^[\p{L}\p{M}\p{N}\s.'’\-/,&()]+$/u;

  const normalize = (text) => String(text || "")
    .toLocaleLowerCase("id-ID")
    .replace(/[_\-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();

  const fields = Array.from(form.querySelectorAll('input[name^="field-"], select[name^="field-"], textarea[name^="field-"]'));
  const findLabel = (field) => Array.from(form.querySelectorAll('label[for]')).find(label => label.htmlFor === field.id);

  function typeOfField(field) {
    const key = normalize(field.name.replace(/^field-/, ""));
    const label = normalize(findLabel(field)?.textContent).replace(/\s*16 digit\s*$/, "");
    const identity = `${key} ${label}`;

    if (/(^|\s)nik($|\s)/.test(identity)) return "nik";
    if (/(^|\s)agama($|\s)/.test(identity)) return "religion";
    if (/^(nomor surat|no surat|nomor urut surat|nomor urut|no urut)$/.test(key)
      || /^(nomor surat|no\.? surat|nomor urut surat|nomor urut)$/.test(label)) return "letter-number";
    if (/(^|\s)tempat lahir($|\s)|(^|\s)kota lahir($|\s)/.test(identity)) return "birthplace";
    if (/(^|\s)pekerjaan($|\s)|(^|\s)profesi($|\s)/.test(identity)) return "occupation";
    if ((/(^|\s)nama lengkap($|\s)|(^|\s)nama pemohon($|\s)/.test(identity) || key === "nama")
      && !/(usaha|toko|perusahaan|lembaga|instansi|produk)/.test(identity)) return "full-name";
    return "other";
  }

  function asReligionSelect(original) {
    if (original.tagName === "SELECT") return original;
    if (original.tagName !== "INPUT") return original;

    const select = document.createElement("select");
    for (const attribute of ["id", "name", "class", "aria-label", "aria-describedby", "required", "disabled"]) {
      if (original.hasAttribute(attribute)) select.setAttribute(attribute, original.getAttribute(attribute));
    }
    const initial = original.value.trim();
    const prompt = document.createElement("option");
    prompt.value = "";
    prompt.textContent = "Pilih agama";
    select.appendChild(prompt);
    RELIGIONS.forEach(value => {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = value;
      select.appendChild(option);
    });

    // Nilai lama yang berbeda dari enam opsi resmi tetap dipertahankan,
    // agar tidak berubah diam-diam ketika pengguna menyimpan field lain.
    if (initial && !RELIGIONS.includes(initial)) {
      const legacyOption = document.createElement("option");
      legacyOption.value = initial;
      legacyOption.textContent = `${initial} (nilai sebelumnya)`;
      select.appendChild(legacyOption);
    }
    select.value = initial;
    original.replaceWith(select);
    return select;
  }

  const controls = fields.map(original => {
    const rule = typeOfField(original);
    const field = rule === "religion" ? asReligionSelect(original) : original;
    field.dataset.validationRule = rule;

    if (rule === "nik") {
      field.type = "text"; // bukan number: nol di awal NIK tidak boleh hilang
      field.inputMode = "numeric";
      field.autocomplete = "off";
      field.maxLength = 16;
      field.pattern = "[0-9]{16}";
    } else if (rule === "letter-number" && field.tagName === "INPUT") {
      // Ini bagian setelah "100/" pada nomor surat, BUKAN nomor urut 100.
      // Nilai dapat kosong, berupa huruf, angka, atau gabungannya.
      field.type = "text";
      field.inputMode = "text";
      field.removeAttribute("pattern");
      field.placeholder = "Opsional — boleh kosong, huruf, atau angka";
    }
    return field;
  });

  function messageFor(field) {
    field.setCustomValidity("");
    const value = field.value.trim();
    const rule = field.dataset.validationRule;

    // Partial save tetap diizinkan seperti sebelum validasi dipasang.
    // Kelengkapan semua field untuk status "Selesai" masih diperiksa backend.
    if (!value) return field.required ? "Kolom ini wajib diisi." : "";

    switch (rule) {
      case "nik":
        return /^[0-9]{16}$/.test(value) ? "" : "NIK harus tepat 16 digit angka (tanpa spasi atau tanda baca).";
      case "letter-number":
        // Bagian ini opsional, jadi tidak mensyaratkan angka ataupun huruf.
        return "";
      case "full-name":
        return NAME_CHARS.test(value) && LETTER.test(value)
          ? "" : "Nama lengkap hanya boleh berisi huruf, spasi, titik, apostrof, atau tanda hubung.";
      case "birthplace":
        return PLACE_CHARS.test(value) && LETTER.test(value)
          ? "" : "Tempat lahir harus memuat nama tempat; angka dan tanda baca umum boleh jika bagian dari nama wilayah.";
      case "occupation":
        return JOB_CHARS.test(value) && LETTER.test(value)
          ? "" : "Pekerjaan harus memuat huruf; angka dan tanda baca umum boleh untuk nama profesi.";
      case "religion":
        return RELIGIONS.includes(value) || Array.from(field.options).some(opt => opt.value === value && opt.textContent.includes("nilai sebelumnya"))
          ? "" : "Silakan pilih agama dari daftar.";
      default:
        if (field.validity.badInput || field.validity.typeMismatch || field.validity.rangeOverflow || field.validity.rangeUnderflow || field.validity.stepMismatch || field.validity.patternMismatch) {
          return "Format isian belum sesuai. Periksa kembali data yang dimasukkan.";
        }
        return "";
    }
  }

  function errorNode(field) {
    const id = `${field.id}-validation-error`;
    let node = document.getElementById(id);
    if (!node) {
      node = document.createElement("small");
      node.id = id;
      node.className = "identity-field-error";
      node.setAttribute("aria-live", "polite");
      node.hidden = true;
      field.insertAdjacentElement("afterend", node);
    }
    return node;
  }

  function validate(field, show) {
    const message = messageFor(field);
    field.setCustomValidity(message);
    const node = errorNode(field);
    const shouldDisplay = Boolean(message && show);
    node.textContent = shouldDisplay ? message : "";
    node.hidden = !shouldDisplay;
    field.classList.toggle("identity-invalid", shouldDisplay);
    if (shouldDisplay) {
      field.setAttribute("aria-invalid", "true");
      const ids = new Set((field.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean));
      ids.add(node.id);
      field.setAttribute("aria-describedby", Array.from(ids).join(" "));
    } else {
      field.removeAttribute("aria-invalid");
      const ids = (field.getAttribute("aria-describedby") || "").split(/\s+/).filter(id => id && id !== node.id);
      if (ids.length) field.setAttribute("aria-describedby", ids.join(" "));
      else field.removeAttribute("aria-describedby");
    }
    return !message;
  }

  controls.forEach(field => {
    field.addEventListener("blur", () => {
      field.dataset.validationTouched = "true";
      validate(field, true);
    });
    field.addEventListener("input", () => {
      if (field.dataset.validationTouched === "true") validate(field, true);
    });
    field.addEventListener("change", () => {
      if (field.dataset.validationTouched === "true") validate(field, true);
    });
  });

  form.noValidate = true; // pesan invalid konsisten, bukan popup bawaan browser
  form.addEventListener("submit", event => {
    const invalid = controls.filter(field => {
      field.dataset.validationTouched = "true";
      return !validate(field, true);
    });
    let summary = form.querySelector(".identity-validation-summary");
    if (invalid.length) {
      event.preventDefault();
      if (!summary) {
        summary = document.createElement("p");
        summary.className = "identity-validation-summary";
        summary.setAttribute("role", "alert");
        const firstLabel = form.querySelector('label[for]');
        if (firstLabel) form.insertBefore(summary, firstLabel);
        else form.prepend(summary);
      }
      summary.textContent = `Ada ${invalid.length} kolom dengan format tidak sesuai. Periksa pesan merah di bawah kolom.`;
      invalid[0].focus({ preventScroll: true });
      invalid[0].scrollIntoView({ behavior: "smooth", block: "center" });
    } else if (summary) {
      summary.remove();
    }
  });
})();
