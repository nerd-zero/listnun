<template>
  <Transition
    name="ln-collapse"
    @before-enter="toZero"
    @enter="toContent"
    @after-enter="release"
    @before-leave="toContent"
    @leave="collapse"
    @after-leave="release"
  >
    <div v-if="open" class="ln-collapse">
      <slot />
    </div>
  </Transition>
</template>

<script setup lang="ts">
// Shows its content with a height slide, like v-if but animated. The
// content is unmounted while closed, so hidden required fields can't block
// a form. Timing and easing are in assets/motion.scss.
defineProps<{ open: boolean }>();

function px(el: Element): string {
  return `${(el as HTMLElement).scrollHeight}px`;
}

function toZero(el: Element) {
  const h = el as HTMLElement;
  h.style.height = '0';
}

function toContent(el: Element) {
  const h = el as HTMLElement;
  // Read layout first so the browser commits the starting height before
  // the change, or it skips the transition.
  void h.offsetHeight; // eslint-disable-line no-void
  h.style.height = px(h);
}

function collapse(el: Element) {
  const h = el as HTMLElement;
  void h.offsetHeight; // eslint-disable-line no-void
  h.style.height = '0';
}

function release(el: Element) {
  const h = el as HTMLElement;
  h.style.height = '';
}
</script>
