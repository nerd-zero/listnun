import { ref, type Ref } from 'vue';

const STORAGE_KEY = 'lm-dark';
const DARK_CLASS = 'app-dark';
// Added to <html> for one switch so colours cross-fade (motion.scss).
const SWITCHING_CLASS = 'ln-theme-switching';
// A little longer than --ln-dur-slow, so the cross-fade finishes before the
// class (and its transition on every element) is removed.
const SWITCH_MS = 300;
let switchTimer: number | undefined;

// The app's theme mode lives on <html> as a class (the PrimeVue preset's
// darkModeSelector points at it), and the email builder needs the same
// mode. The state is kept here, module-wide, so any component can read it
// without prop drilling and without watching the DOM.
const isDark: Ref<boolean> = ref(localStorage.getItem(STORAGE_KEY) === '1');

if (isDark.value) {
  document.documentElement.classList.add(DARK_CLASS);
}

function apply(dark: boolean) {
  const html = document.documentElement;
  html.classList.add(SWITCHING_CLASS);
  html.classList.toggle(DARK_CLASS, dark);
  window.clearTimeout(switchTimer);
  switchTimer = window.setTimeout(() => html.classList.remove(SWITCHING_CLASS), SWITCH_MS);
  localStorage.setItem(STORAGE_KEY, dark ? '1' : '');
}

// useDarkMode returns the app's theme mode and a toggle for it.
export function useDarkMode() {
  function toggleDark() {
    isDark.value = !isDark.value;
    apply(isDark.value);
  }

  function setDark(dark: boolean) {
    if (dark === isDark.value) return;
    isDark.value = dark;
    apply(dark);
  }

  return { isDark, toggleDark, setDark };
}

export default useDarkMode;
