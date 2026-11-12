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
    let activeFilter = 'all';
    const redraw = () => {
      const q = (search?.value || '').trim().toLocaleLowerCase('id');
      let visible = 0;
      rows.forEach(row => {
        const isActive = row.dataset.active === 'true';
        const matchedStatus = activeFilter === 'all' || (activeFilter === 'active' ? isActive : !isActive);
        const shown = matchedStatus && (row.dataset.search || '').toLocaleLowerCase('id').includes(q);
        row.hidden = !shown;
        if (shown) visible++;
      });
      if (count) count.textContent = String(visible);
      if (noResults) noResults.hidden = visible !== 0;
    };
    search?.addEventListener('input', redraw);
    view.querySelector('[data-search-trigger]')?.addEventListener('click', () => search?.focus());
    filters.forEach(button => button.addEventListener('click', () => {
      activeFilter = button.dataset.filter;
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
      if (!form || !window.confirm('Hapus FAQ ini secara permanen? Tindakan ini tidak dapat dibatalkan.\n\nUntuk menyembunyikan sementara, gunakan Edit lalu hapus centang "Aktif".')) return;
      form.requestSubmit ? form.requestSubmit() : form.submit();
      return;
    }
    if (event.target instanceof HTMLDialogElement) {
      const dialog = event.target;
      const rect = dialog.getBoundingClientRect();
      if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
    }
  });
})();
