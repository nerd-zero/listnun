<template>
  <section class="log-view">
    <div v-if="loading" class="log-view-loading">
      <PvProgressSpinner style="width:2rem;height:2rem" />
    </div>
    <ol class="log-view-lines">
      <li v-for="(line, i) in parsed" :key="i" class="log-view-line">
        <span v-if="line.timestamp" class="log-view-time">{{ line.timestamp }}</span>
        <span v-if="line.file && line.file !== '*'" class="log-view-file">{{ line.file }}:</span>
        <span class="log-view-msg">{{ line.message }}</span>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';

// LogView renders server log lines inside a dark .ln-code panel. It doesn't
// scroll itself; the panel around it does (and keeps itself at the bottom).

// Splits "2021/05/01 00:00:00.000000 init.go:99: message".
const reFormatLine = /^([0-9\s:/]+\.[0-9]{6}) (.+?\.go:[0-9]+|\*):\s(.+)$/;

const props = withDefaults(defineProps<{
  loading?: boolean;
  lines?: unknown[];
}>(), {
  loading: false,
  lines: () => [],
});

function splitLine(l: string) {
  const parts = l.split(reFormatLine);
  if (parts.length !== 5) {
    return { timestamp: '', file: '', message: l };
  }
  return { timestamp: parts[1], file: parts[2], message: parts[3] };
}

const parsed = computed(() => props.lines.filter(Boolean).map((l) => splitLine(String(l))));
</script>

<style scoped>
/* Colours are fixed, like .ln-code's own: the panel is dark in both themes. */
.log-view-loading { display: flex; justify-content: center; padding: 24px; }
.log-view-lines { list-style: none; margin: 0; padding: 0; }
.log-view-line {
  display: flex;
  gap: 12px;
  padding: 1px 6px;
  margin: 0 -6px;
  border-radius: 4px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.log-view-line:hover { background: rgba(255, 255, 255, .05); }
.log-view-time, .log-view-file { flex: none; color: #9599a0; }
.log-view-msg { color: #ece9e1; }
</style>
