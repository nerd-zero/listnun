// Sign-in pages: show/hide password buttons and the new-password strength
// meter. No dependencies; the forms work without it.
(function () {
  document.querySelectorAll('[data-toggle-password]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var input = document.getElementById(btn.getAttribute('data-toggle-password'));
      if (!input) return;
      var show = input.type === 'password';
      input.type = show ? 'text' : 'password';
      btn.setAttribute('aria-pressed', show ? 'true' : 'false');
    });
  });

  // score is 0-4: length 8+, length 12+, mixed case, digits or symbols.
  function score(v) {
    if (!v) return 0;
    var s = 0;
    if (v.length >= 8) s += 1;
    if (v.length >= 12) s += 1;
    if (/[a-z]/.test(v) && /[A-Z]/.test(v)) s += 1;
    if (/[0-9]/.test(v) || /[^A-Za-z0-9]/.test(v)) s += 1;
    return v.length < 8 ? Math.min(s, 1) : s;
  }

  // Language picker: remember the choice for a year and reload. The admin
  // app reads the same cookie (ln_lang).
  document.querySelectorAll('[data-lang-picker]').forEach(function (sel) {
    sel.addEventListener('change', function () {
      var maxAge = 60 * 60 * 24 * 365;
      document.cookie = 'ln_lang=' + encodeURIComponent(sel.value) + '; path=/; max-age=' + maxAge + '; samesite=lax';
      window.location.reload();
    });
  });

  document.querySelectorAll('[data-strength]').forEach(function (input) {
    var meter = document.getElementById(input.getAttribute('data-strength'));
    if (!meter) return;
    var label = meter.nextElementSibling;
    var words = (meter.getAttribute('data-words') || '').split('|');
    input.addEventListener('input', function () {
      var s = score(input.value);
      meter.setAttribute('data-score', String(s));
      if (label) {
        label.textContent = input.value ? (words[s] || '') : (words[0] || '');
        label.classList.toggle('is-ok', s >= 3);
      }
    });
  });
})();
