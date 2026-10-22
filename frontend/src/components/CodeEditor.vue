<template>
  <div ref="editorEl" class="code-editor" />
</template>

<script setup lang="ts">
import {
  ref, watch, onMounted, onBeforeUnmount, nextTick,
} from 'vue';
import { EditorState } from '@codemirror/state';
import {
  EditorView, keymap, highlightActiveLine, lineNumbers, highlightActiveLineGutter,
} from '@codemirror/view';
import { markdown } from '@codemirror/lang-markdown';
import { javascript } from '@codemirror/lang-javascript';
import { css } from '@codemirror/lang-css';
import { html } from '@codemirror/lang-html';
import {
  defaultKeymap, history, historyKeymap, indentWithTab,
} from '@codemirror/commands';
import { defaultHighlightStyle, syntaxHighlighting, bracketMatching } from '@codemirror/language';
import { search, searchKeymap, highlightSelectionMatches } from '@codemirror/search';
import { vsCodeLight, listnunInk } from './editor-theme';

const props = withDefaults(defineProps<{
  modelValue?: string;
  lang?: string;
  disabled?: boolean;
  // 'ink' is the dark code surface used in the Developers area.
  theme?: 'light' | 'ink';
  autofocus?: boolean;
}>(), {
  modelValue: '',
  lang: 'html',
  disabled: false,
  theme: 'light',
  autofocus: true,
});

const emit = defineEmits(['update:modelValue']);

const editorEl = ref<HTMLElement | null>(null);
let editor: EditorView | null = null;
let internalUpdate = false;

onMounted(() => {
  const onUpdate = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      internalUpdate = true;
      emit('update:modelValue', update.state.doc.toString());
    }
  });

  let langs: any[] = [];
  switch (props.lang) {
    case 'html': langs = [html()]; break;
    case 'css': langs = [css()]; break;
    case 'javascript': langs = [javascript()]; break;
    case 'markdown': langs = [markdown()]; break;
    case 'json': langs = [javascript()]; break;
    default: langs = [html()];
  }

  const stateCfg = EditorState.create({
    doc: props.modelValue,
    extensions: [
      EditorView.baseTheme({}),
      ...langs,
      history(),
      highlightActiveLine(),
      bracketMatching(),
      highlightSelectionMatches(),
      lineNumbers(),
      highlightActiveLineGutter(),
      keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, indentWithTab]),
      EditorState.readOnly.of(props.disabled),
      EditorView.editable.of(!props.disabled),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      EditorView.lineWrapping,
      props.theme === 'ink' ? listnunInk : vsCodeLight,
      search({ top: true }),
      onUpdate,
    ],
  });

  editor = new EditorView({ state: stateCfg, parent: editorEl.value! });

  if (props.autofocus) {
    nextTick(() => {
      window.setTimeout(() => { editor?.focus(); }, 100);
    });
  }
});

onBeforeUnmount(() => {
  editor?.destroy();
});

watch(
  () => props.modelValue,
  (val) => {
    if (!internalUpdate && editor) {
      editor.dispatch({
        changes: { from: 0, to: editor.state.doc.length, insert: val },
      });
    }
    internalUpdate = false;
  },
);
</script>
