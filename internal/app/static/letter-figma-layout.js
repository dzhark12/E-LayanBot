/* Layout-only enhancement: form values, names, methods and endpoints are untouched. */
(() => {
  const order = [
    ['nomor surat','nomor surat'], ['penandatangan','penandatangan surat'],
    ['nama lengkap','nama lengkap'], ['jenis kelamin','jenis kelamin'],
    ['tempat lahir','tempat lahir'], ['tanggal lahir','tanggal lahir'],
    ['nik','nik'], ['agama','agama'],
    ['status perkawinan','status perkawinan'], ['pekerjaan','pekerjaan'],
    ['alamat','alamat'], ['kecamatan','kecamatan'], ['alamat domisili','alamat domisili'],
    ['jenis usaha','jenis usaha'], ['lokasi usaha','lokasi usaha']
  ];
  const clean = str => str.toLowerCase().replace(/\s*16 digit\s*/g,'').replace(/\s+/g,' ').trim();
  document.querySelectorAll('.figma-letter-detail .identity-form').forEach(form => {
    const labels = Array.from(form.children).filter(el => el.tagName === 'LABEL' && el.hasAttribute('for'));
    if (!labels.length) return;
    const fields = labels.map((label, idx) => {
      const control = label.nextElementSibling;
      if (!control || !/^(INPUT|SELECT|TEXTAREA)$/.test(control.tagName)) return null;
      const text=clean(label.textContent);
      const wrapper=document.createElement('div');
      wrapper.className='figma-form-field';
      wrapper.dataset.fieldName=text;
      wrapper.append(label,control);
      if (text === 'alamat' || text.includes('alamat lengkap')) wrapper.classList.add('field-wide');
      if (text === 'nik') wrapper.classList.add('field-nik');
      return {wrapper, text, idx};
    }).filter(Boolean);
    if (!fields.length) return;
    const rank = item => {
      const n=item.text;
      const i=order.findIndex(([key]) => n===key || (key==='penandatangan' && n.includes('penandatangan')));
      return i<0?100+item.idx:i;
    };
    fields.sort((a,b)=>rank(a)-rank(b));
    const grid=document.createElement('div'); grid.className='figma-fields-grid';
    let headingAdded=false;
    let businessAdded=false;
    fields.forEach(item=>{
      const n=item.text;
      if (!headingAdded && rank(item)>=2){
        const heading=document.createElement('div'); heading.className='figma-section-label';heading.textContent='DATA PEMOHON'; grid.append(heading);headingAdded=true;
      }
      if(!businessAdded && (n.includes('jenis usaha') || n.includes('lokasi usaha'))){
        const heading=document.createElement('div'); heading.className='figma-section-label';heading.textContent='DATA USAHA';grid.append(heading);businessAdded=true;
      }
      if(rank(item)<2) item.wrapper.classList.add('field-wide');
      if(n==='kecamatan' && !document.body.classList.contains('figma-regenerate-surat')) item.wrapper.classList.add('field-wide');
      grid.append(item.wrapper);
    });
    const submit=form.querySelector('button[type=submit]');
    if(submit) form.insertBefore(grid,submit);
    else form.append(grid);
  });
})();
