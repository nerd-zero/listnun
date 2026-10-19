import { onBeforeUnmount, ref } from 'vue';

// useMedia tracks a CSS media query, e.g. useMedia('(max-width: 767px)').
// eslint-disable-next-line import/prefer-default-export
export function useMedia(query: string) {
  const mql = window.matchMedia(query);
  const matches = ref(mql.matches);
  const onChange = (e: MediaQueryListEvent) => { matches.value = e.matches; };
  mql.addEventListener('change', onChange);
  onBeforeUnmount(() => mql.removeEventListener('change', onChange));
  return matches;
}
