<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-for="(path, index) in paths" :key="index" :d="path" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  size: { type: Number, default: 16 },
})

const icons = {
  add: ['M12 5v14', 'M5 12h14'],
  chevronDown: ['m6 9 6 6 6-6'],
  chevronRight: ['m9 18 6-6-6-6'],
  clock: ['M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z', 'M12 6v6l4 2'],
  copy: ['M8 8h11v11H8z', 'M5 16H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h11a1 1 0 0 1 1 1v1'],
  delete: ['M3 6h18', 'M8 6V4h8v2', 'm19 6-1 15H6L5 6', 'M10 11v5', 'M14 11v5'],
  file: ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z', 'M14 2v6h6'],
  flow: ['M6 3v12', 'M18 9v12', 'M6 15c0 3 2 4 5 4h7', 'M3 6l3-3 3 3', 'm15 18 3 3 3-3'],
  folder: ['M3 6a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z'],
  history: ['M3 12a9 9 0 1 0 3-6.7L3 8', 'M3 3v5h5', 'M12 7v5l3 2'],
  layout: ['M4 4h6v16H4z', 'M14 4h6v7h-6z', 'M14 15h6v5h-6z'],
  more: ['M5 12h.01', 'M12 12h.01', 'M19 12h.01'],
  pause: ['M8 5v14', 'M16 5v14'],
  play: ['m8 5 11 7-11 7Z'],
  refresh: ['M20 11a8.1 8.1 0 0 0-15.5-2M4 4v5h5', 'M4 13a8.1 8.1 0 0 0 15.5 2M20 20v-5h-5'],
  save: ['M5 3h12l2 2v16H5Z', 'M8 3v6h8V3', 'M8 21v-7h8v7'],
  search: ['m21 21-4.35-4.35', 'M11 18a7 7 0 1 1 0-14 7 7 0 0 1 0 14Z'],
  server: ['M4 4h16v6H4z', 'M4 14h16v6H4z', 'M8 7h.01', 'M8 17h.01'],
  settings: ['M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z', 'M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.12 2.12-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.02 1.55V20H9.72v-.09a1.7 1.7 0 0 0-1.02-1.55 1.7 1.7 0 0 0-1.88.34l-.06.06-2.12-2.12.06-.06A1.7 1.7 0 0 0 5.04 15a1.7 1.7 0 0 0-1.55-1.02H3v-3h.49A1.7 1.7 0 0 0 5.04 10a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.12-2.12.06.06A1.7 1.7 0 0 0 8.7 6.34a1.7 1.7 0 0 0 1.02-1.55V4.7h5v.09a1.7 1.7 0 0 0 1.02 1.55 1.7 1.7 0 0 0 1.88-.34l.06-.06 2.12 2.12-.06.06A1.7 1.7 0 0 0 19.4 10a1.7 1.7 0 0 0 1.55 1.02H21v3h-.05A1.7 1.7 0 0 0 19.4 15Z'],
  stop: ['M7 7h10v10H7z'],
  terminal: ['m4 17 6-6-6-6', 'M12 19h8'],
  x: ['M18 6 6 18', 'm6 6 12 12'],
}

const paths = computed(() => icons[props.name] || icons.file)
</script>
