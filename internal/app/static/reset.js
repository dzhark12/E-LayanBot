// E-LAYAN — interaksi halaman reset kata sandi baru.
(function () {
  'use strict';

  function setupToggle(toggleBtnSelector, inputId) {
    var toggleBtn = document.querySelector(toggleBtnSelector);
    var input = document.getElementById(inputId);
    if (!toggleBtn || !input) return;

    var iconShow = toggleBtn.querySelector('.icon-eye');
    var iconHide = toggleBtn.querySelector('.icon-eye-off');

    toggleBtn.addEventListener('click', function () {
      var isPassword = input.getAttribute('type') === 'password';
      input.setAttribute('type', isPassword ? 'text' : 'password');
      if (iconShow) iconShow.classList.toggle('hidden', isPassword);
      if (iconHide) iconHide.classList.toggle('hidden', !isPassword);
      toggleBtn.setAttribute('aria-pressed', isPassword ? 'true' : 'false');
    });
  }

  setupToggle('.toggle-pass1', 'password');
  setupToggle('.toggle-pass2', 'password_confirm');

  // Indikator kekuatan password & konfirmasi
  var passInput = document.getElementById('password');
  var confirmInput = document.getElementById('password_confirm');
  var fill = document.getElementById('strength-fill');
  var label = document.getElementById('strength-label');
  var charCount = document.getElementById('char-count');
  var matchHint = document.getElementById('match-hint');

  function evaluateStrength(val) {
    if (!fill || !label || !charCount) return;
    var len = val.length;
    charCount.textContent = len + ' / 12';

    fill.className = 'strength-fill';
    if (len === 0) {
      fill.style.width = '0%';
      label.textContent = 'Minimal 12 karakter';
      label.style.color = '#64748b';
    } else if (len < 12) {
      fill.classList.add('strength-weak');
      label.textContent = 'Kurang panjang (minimal 12 karakter)';
      label.style.color = '#ef4444';
    } else {
      var hasLetters = /[a-zA-Z]/.test(val);
      var hasDigits = /[0-9]/.test(val);
      var hasSpecial = /[^a-zA-Z0-9]/.test(val);

      if (len >= 14 && hasLetters && hasDigits && hasSpecial) {
        fill.classList.add('strength-strong');
        label.textContent = 'Sangat Kuat';
        label.style.color = '#15803d';
      } else if (len >= 12 && hasLetters && (hasDigits || hasSpecial)) {
        fill.classList.add('strength-medium');
        label.textContent = 'Kuat (Memenuhi Syarat)';
        label.style.color = '#f59e0b';
      } else {
        fill.classList.add('strength-medium');
        label.textContent = 'Cukup (Disarankan tambah kombinasi angka/simbol)';
        label.style.color = '#f59e0b';
      }
    }
  }

  function checkMatch() {
    if (!confirmInput || !passInput || !matchHint) return;
    var pass = passInput.value;
    var conf = confirmInput.value;
    if (conf.length === 0) {
      matchHint.textContent = '';
      return;
    }
    if (pass === conf) {
      matchHint.textContent = '✓ Kata sandi cocok';
      matchHint.style.color = '#15803d';
    } else {
      matchHint.textContent = '✗ Konfirmasi kata sandi belum cocok';
      matchHint.style.color = '#ef4444';
    }
  }

  if (passInput) {
    passInput.addEventListener('input', function () {
      evaluateStrength(passInput.value);
      checkMatch();
    });
  }

  if (confirmInput) {
    confirmInput.addEventListener('input', checkMatch);
  }

  var form = document.getElementById('reset-form');
  var submitBtn = document.getElementById('submit-button');
  if (form && submitBtn) {
    form.addEventListener('submit', function () {
      submitBtn.disabled = true;
      submitBtn.classList.add('loading');
      var text = submitBtn.querySelector('.login-btn-text');
      if (text) text.textContent = 'Menyimpan Kata Sandi...';
    });
  }
})();

