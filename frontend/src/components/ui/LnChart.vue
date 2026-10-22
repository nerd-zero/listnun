<template>
  <div class="ln-chart" :style="{ height }">
    <canvas ref="canvas" :aria-label="label" role="img" />
  </div>
</template>

<script setup lang="ts">
import {
  onBeforeUnmount, onMounted, ref, watch,
} from 'vue';
import Chart from 'chart.js/auto';
import type { ChartConfiguration } from 'chart.js';

export interface ChartSeries {
  label: string;
  data: number[];
  // A token name: success, warn, danger, neutral or ink.
  tone: 'success' | 'warn' | 'danger' | 'neutral' | 'ink';
}

// LnChart: chart.js on the tokens board's rules: series colours from the
// --ln-* variables, gridlines off, one axis line, no in-canvas legend (the
// card shows one above). It redraws when the theme changes.
const props = withDefaults(defineProps<{
  type: 'bar' | 'line';
  labels: string[];
  series: ChartSeries[];
  stacked?: boolean;
  // Fills the area under a single line series.
  area?: boolean;
  height?: string;
  label?: string;
}>(), {
  stacked: false,
  area: false,
  height: '220px',
  label: '',
});

const canvas = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;
let observer: MutationObserver | null = null;

const toneVar: Record<ChartSeries['tone'], string> = {
  success: '--ln-success',
  warn: '--ln-warn',
  danger: '--ln-danger',
  neutral: '--ln-neutral',
  ink: '--ln-ink',
};

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function config(): ChartConfiguration {
  const text3 = cssVar('--ln-text-3');
  const border = cssVar('--ln-border');
  const muted = cssVar('--ln-muted');

  return {
    type: props.type,
    data: {
      labels: props.labels,
      datasets: props.series.map((s) => {
        const color = cssVar(toneVar[s.tone]);
        return {
          label: s.label,
          data: s.data,
          backgroundColor: props.type === 'line' ? muted : color,
          borderColor: color,
          borderWidth: props.type === 'line' ? 2 : 0,
          borderRadius: props.type === 'bar' ? 2 : 0,
          maxBarThickness: 22,
          fill: props.area,
          pointRadius: 0,
          pointHoverRadius: 4,
          tension: 0.25,
        };
      }),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      interaction: { intersect: false, mode: 'index' },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: cssVar('--ln-card'),
          titleColor: cssVar('--ln-text'),
          bodyColor: cssVar('--ln-text-2'),
          borderColor: border,
          borderWidth: 1,
          padding: 10,
          boxPadding: 4,
        },
      },
      scales: {
        x: {
          stacked: props.stacked,
          grid: { display: false },
          border: { color: border },
          ticks: {
            color: text3,
            font: { size: 12 },
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 3,
          },
        },
        y: {
          stacked: props.stacked,
          display: false,
          beginAtZero: props.type === 'bar',
        },
      },
    },
  } as ChartConfiguration;
}

function render() {
  if (!canvas.value) return;
  chart?.destroy();
  chart = new Chart(canvas.value, config());
}

watch(() => [props.labels, props.series], render, { deep: true });

onMounted(() => {
  render();
  // Redraw with the new token colours when dark mode is toggled.
  observer = new MutationObserver(render);
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
});

onBeforeUnmount(() => {
  observer?.disconnect();
  chart?.destroy();
});
</script>

<style scoped>
.ln-chart { position: relative; width: 100%; }
</style>
