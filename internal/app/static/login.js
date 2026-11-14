// E-LAYAN — interaksi halaman login.
// Sengaja dipisah dari login.html: skrip inline dan atribut onclick bisa diblokir
// oleh header Content-Security-Policy (script-src 'self'), file eksternal tidak.
(function () {
  'use strict';

  // 1. Tampilkan / sembunyikan kata sandi
  var passwordInput = document.getElementById('password');
  var toggleBtn = document.querySelector('.form-input-toggle');
  var iconShow = toggleBtn && toggleBtn.querySelector('.icon-eye');
  var iconHide = toggleBtn && toggleBtn.querySelector('.icon-eye-off');

  if (toggleBtn && passwordInput && iconShow && iconHide) {
    toggleBtn.addEventListener('click', function () {
      var isPassword = passwordInput.getAttribute('type') === 'password';
      passwordInput.setAttribute('type', isPassword ? 'text' : 'password');
      iconShow.classList.toggle('hidden', isPassword);
      iconHide.classList.toggle('hidden', !isPassword);
      toggleBtn.setAttribute('aria-pressed', isPassword ? 'true' : 'false');
    });
  }

  // 2. Tutup pesan error dari server
  var errorAlert = document.querySelector('.form-error-alert');
  var closeBtn = errorAlert && errorAlert.querySelector('.form-error-close');
  if (closeBtn && errorAlert) {
    closeBtn.addEventListener('click', function () {
      errorAlert.classList.add('is-closing');
      setTimeout(function () {
        errorAlert.remove();
      }, 200);
    });
  }

  // 3. Status loading saat form login dikirim.
  var form = document.getElementById('login-form');
  var submitBtn = document.getElementById('submit-button');
  var spinner = submitBtn && submitBtn.querySelector('.login-spinner');
  var btnText = submitBtn && submitBtn.querySelector('.login-btn-text');
  var btnIcon = submitBtn && submitBtn.querySelector('.login-btn-icon');

  if (form && submitBtn && spinner && btnText && btnIcon) {
    form.addEventListener('submit', function () {
      submitBtn.disabled = true;
      submitBtn.classList.add('loading');
      spinner.classList.remove('hidden');
      btnIcon.classList.add('hidden');
      btnText.textContent = 'Memverifikasi Data...';
    });
  }
})();
