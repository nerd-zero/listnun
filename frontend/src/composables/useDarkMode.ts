import { ref, type Ref } from 'vue';

const STORAGE_KEY = 'lm-dark';
const DARK_CLASS = 'app-dark';

// The app's theme mode lives on <html> as a class (the PrimeVue preset's
// darkModeSelector points at it), and the email builder needs the same
// mode. The state is kept here, module-wide, so any component can read it
// without prop drilling and without watching the DOM.
const isDark: Ref<boolean> = ref(localStorage.getItem(STORAGE_KEY) === '1');

if (isDark.value) {
  document.documentElement.classList.add(DARK_CLASS);
}

// useDarkMode returns the app's theme mode and a toggle for it.
export function useDarkMode() {
  function toggleDark() {
    isDark.value = !isDark.value;
    document.documentElement.classList.toggle(DARK_CLASS, isDark.value);
    localStorage.setItem(STORAGE_KEY, isDark.value ? '1' : '');
  }

  function setDark(dark: boolean) {
    if (dark === isDark.value) return;
    isDark.value = dark;
    document.documentElement.classList.toggle(DARK_CLASS, dark);
    localStorage.setItem(STORAGE_KEY, dark ? '1' : '');
  }

  return { isDark, toggleDark, setDark };
}

export default useDarkMode;
