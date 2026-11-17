// Antarmuka tabel layanan/FAQ. Tidak mengganti endpoint backend atau metode POST.
(() => {
  'use strict';
  const views = [...document.querySelectorAll('.workspace-view[data-view]')];
  if (!views.length) return;

  const showView = () => {
    const current = location.hash.toLowerCase() === '#faq' ? 'faq' : 'letters';
    document.body.classList.add('settings-ui-ready');
    views.forEach(view => { view.hidden = view.dataset.view !== current; });
    document.querySelectorAll('[data-workspace-link]').forEach(link => {
      const active = link.dataset.workspaceLink === current;
      link.classList.toggle('active', active);
      if (active) link.setAttribute('aria-current', 'page');
      else link.removeAttribute('aria-current');
    });
    document.title = (current === 'faq' ? 'FAQ bot' : 'Data layanan') + ' · E-LAYAN Desa Sabah Balau';
  };
  window.addEventListener('hashchange', showView);
  showView();

  views.forEach(view => {
    const rows = [...view.querySelectorAll('[data-record]')];
    const search = view.querySelector('[data-table-search]');
    const filters = [...view.querySelectorAll('[data-filter]')];
    const count = view.querySelector('[data-visible-count]');
    const noResults = view.querySelector('[data-no-results]');
    const pagination = view.querySelector('[data-table-pagination]');
    const pageSize = 10;
    let currentPage = 1;
    let activeFilter = 'all';
    const redraw = () => {
      const q = (search?.value || '').trim().toLocaleLowerCase('id');
      const matchingRows = rows.filter(row => {
        const isActive = row.dataset.active === 'true';
        const matchedStatus = activeFilter === 'all' || (activeFilter === 'active' ? isActive : !isActive);
        return matchedStatus && (row.dataset.search || '').toLocaleLowerCase('id').includes(q);
      });
      const pageCount = Math.ceil(matchingRows.length / pageSize);
      currentPage = Math.min(Math.max(currentPage, 1), Math.max(pageCount, 1));
      const start = (currentPage - 1) * pageSize;
      const pageRows = new Set(matchingRows.slice(start, start + pageSize));
      rows.forEach(row => { row.hidden = !pageRows.has(row); });
      if (count) count.textContent = String(matchingRows.length);
      if (noResults) noResults.hidden = matchingRows.length !== 0;
      if (!pagination) return;

      pagination.replaceChildren();
      pagination.hidden = pageCount <= 1;
      if (pageCount <= 1) return;
      const addButton = (label, page, ariaLabel, disabled = false, current = false) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'faq-page-button';
        button.textContent = label;
        button.disabled = disabled;
        if (current) button.setAttribute('aria-current', 'page');
        else button.setAttribute('aria-label', ariaLabel || `Halaman ${page}`);
        button.addEventListener('click', () => {
          currentPage = page;
          redraw();
          pagination.querySelector('[aria-current="page"]')?.focus();
        });
        pagination.append(button);
      };
      addButton('Sebelumnya', currentPage - 1, 'Halaman sebelumnya', currentPage === 1);
      for (let page = 1; page <= pageCount; page++) {
        addButton(String(page), page, `Halaman ${page}`, false, page === currentPage);
      }
      addButton('Berikutnya', currentPage + 1, 'Halaman berikutnya', currentPage === pageCount);
    };
    search?.addEventListener('input', () => { currentPage = 1; redraw(); });
    view.querySelector('[data-search-trigger]')?.addEventListener('click', () => search?.focus());
    filters.forEach(button => button.addEventListener('click', () => {
      activeFilter = button.dataset.filter;
      currentPage = 1;
      filters.forEach(b => {
        const selected = b === button;
        b.classList.toggle('selected', selected);
        b.setAttribute('aria-pressed', String(selected));
      });
      redraw();
    }));
    const setCount = (key, val) => {
      const el = view.querySelector(`[data-count="${key}"]`);
      if (el) el.textContent = String(val);
    };
    setCount('all', rows.length);
    setCount('active', rows.filter(r => r.dataset.active === 'true').length);
    setCount('inactive', rows.filter(r => r.dataset.active !== 'true').length);
    redraw();
  });

  // Chip editor: mempertahankan textarea dan format baris yang diterima backend.
  document.querySelectorAll('[data-tag-editor]').forEach(editor => {
    const source = editor.querySelector('.tag-source');
    const entry = editor.querySelector('[data-tag-entry]');
    const chips = editor.querySelector('[data-tag-chips]');
    if (!source || !entry || !chips) return;
    let values = source.value.split(/[,\n\r]+/).map(v => v.trim()).filter(Boolean);
    const sync = () => {
      source.value = values.join('\n');
      source.dispatchEvent(new Event('input', {bubbles: false}));
      chips.replaceChildren();
      values.forEach((value, i) => {
        const chip = document.createElement('span');
        chip.className = 'tag-chip';
        chip.append(document.createTextNode(value));
        const remove = document.createElement('button');
        remove.type = 'button';
        remove.className = 'tag-chip-remove';
        remove.setAttribute('aria-label', 'Hapus ' + value);
        remove.textContent = '×';
        remove.addEventListener('click', () => { values.splice(i, 1); sync(); entry.focus(); });
        chip.append(remove);
        chips.append(chip);
      });
    };
    const add = () => {
      const inputs = entry.value.split(/[,\n]+/).map(v => v.trim()).filter(Boolean);
      if (!inputs.length) return;
      values.push(...inputs);
      entry.value = '';
      sync();
    };
    entry.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); add(); }
      if (e.key === 'Backspace' && !entry.value && values.length) { values.pop(); sync(); }
    });
    entry.addEventListener('blur', add);
    editor.closest('form')?.addEventListener('submit', e => {
      add(); sync();
      if (source.hasAttribute('data-required') && !values.length) {
        e.preventDefault();
        entry.setCustomValidity('Isi setidaknya satu kata kunci.');
        entry.reportValidity();
        entry.focus();
      } else entry.setCustomValidity('');
    });
    entry.addEventListener('input', () => entry.setCustomValidity(''));
    sync();
  });

  const openDialog = dialog => {
    if (!dialog) return;
    const opened = document.querySelector('dialog[open]');
    if (opened && opened !== dialog) opened.close?.();
    if (dialog.showModal) dialog.showModal(); else dialog.setAttribute('open', '');
  };
  document.addEventListener('click', event => {
    const opener = event.target.closest('[data-open-dialog]');
    if (opener) { openDialog(document.getElementById(opener.dataset.openDialog)); return; }
    const closer = event.target.closest('[data-close-dialog]');
    if (closer) { const dialog = closer.closest('dialog'); dialog?.close?.(); return; }
    const remove = event.target.closest('[data-delete-faq]');
    if (remove) {
      const form = document.getElementById(remove.dataset.deleteFaq);
      if (!form) return;
      const dialogId = remove.dataset.deleteFaq.replace('faq-delete-', 'faq-confirm-delete-');
      openDialog(document.getElementById(dialogId));
      return;
    }
    if (event.target instanceof HTMLDialogElement) {
      const dialog = event.target;
      const rect = dialog.getBoundingClientRect();
      if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
    }
  });
})();
