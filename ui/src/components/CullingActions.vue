<template>
  <div class="culling-actions" :class="{ disconnected: !connected }">
    <ui-icon
      v-for="item in actions"
      :key="item.action"
      light
      class="culling-icon"
      :class="{ active: statusAction === item.action, busy }"
      size="30"
      :title="`${item.label} · 快捷键 ${item.key}`"
      @click.stop="submit(item.action)"
    >
      {{ item.icon }}
    </ui-icon>
    <ui-icon
      v-if="error"
      light
      class="culling-error"
      size="22"
      :title="error"
    >
      link_off
    </ui-icon>
  </div>
</template>

<script setup>
import { onKeyStroke } from '@vueuse/core';
import { computed, ref, toRefs, watch } from 'vue';

const props = defineProps({
  collectionId: String,
  fileId: [Number, String],
});

const { collectionId, fileId } = toRefs(props);

const actions = [
  { action: 'KEEP_ONE', label: '精选保留', icon: 'star', key: '1' },
  { action: 'KEEP_ARCHIVE', label: '保留但不入选', icon: 'archive', key: '2' },
  { action: 'REJECT_ONE', label: '废片待删', icon: 'delete_outline', key: '3' },
  { action: 'INTENTIONAL_MOTION', label: '有意运动模糊', icon: 'blur_on', key: '4' },
];

const statusAction = ref(null);
const busy = ref(false);
const connected = ref(false);
const error = ref('');
let requestVersion = 0;

const sidecarUrl = computed(() => {
  const local = window.localStorage.getItem('fpcSidecarUrl');
  const configured = import.meta.env.VITE_FPC_SIDECAR_URL;
  return (local || configured || 'http://127.0.0.1:8767').replace(/\/$/, '');
});

function isEditableTarget(target) {
  return !!target?.closest?.('input, textarea, select, [contenteditable="true"]');
}

async function parseResponse(response) {
  let payload = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    throw new Error(payload?.error || `sidecar HTTP ${response.status}`);
  }
  return payload;
}

async function loadStatus() {
  const collection = collectionId.value;
  const id = Number(fileId.value);
  if (!collection || !Number.isInteger(id) || id <= 0) {
    statusAction.value = null;
    return;
  }
  const version = ++requestVersion;
  try {
    const query = new URLSearchParams({
      collection_id: collection,
      file_id: String(id),
    });
    const response = await fetch(`${sidecarUrl.value}/api/culling/status?${query}`, {
      method: 'GET',
      cache: 'no-store',
      credentials: 'omit',
    });
    const payload = await parseResponse(response);
    if (version !== requestVersion) return;
    statusAction.value = payload?.status?.action || null;
    connected.value = true;
    error.value = '';
  } catch (e) {
    if (version !== requestVersion) return;
    connected.value = false;
    error.value = `选片 sidecar 不可用：${e?.message || e}`;
  }
}

async function submit(action) {
  if (busy.value) return;
  const collection = collectionId.value;
  const id = Number(fileId.value);
  if (!collection || !Number.isInteger(id) || id <= 0) return;
  busy.value = true;
  try {
    const response = await fetch(`${sidecarUrl.value}/api/culling/action`, {
      method: 'POST',
      credentials: 'omit',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        collection_id: collection,
        file_id: id,
        action,
      }),
    });
    const payload = await parseResponse(response);
    statusAction.value = payload?.status?.action || action;
    connected.value = true;
    error.value = '';
  } catch (e) {
    connected.value = false;
    error.value = `选片记录失败：${e?.message || e}`;
  } finally {
    busy.value = false;
  }
}

watch([collectionId, fileId], loadStatus, { immediate: true });

for (const item of actions) {
  onKeyStroke([item.key], (event) => {
    if (isEditableTarget(event.target)) return;
    event.preventDefault();
    submit(item.action);
  });
}
</script>

<style scoped>
.culling-actions {
  display: flex;
  align-items: center;
  pointer-events: all;
}

.culling-icon {
  padding: 20px 12px;
  color: rgba(255, 255, 255, 0.78);
  text-shadow: #000 0 0 2px;
  cursor: pointer;
  transition: transform 100ms ease, color 100ms ease, opacity 100ms ease;
}

.culling-icon:hover {
  color: white;
  transform: translateY(-1px);
}

.culling-icon.active {
  color: #ffd54f;
  filter: drop-shadow(0 0 5px rgba(255, 213, 79, 0.55));
}

.culling-icon.busy {
  opacity: 0.45;
  pointer-events: none;
}

.culling-actions.disconnected .culling-icon:not(.active) {
  opacity: 0.55;
}

.culling-error {
  padding: 20px 10px 20px 4px;
  color: #ff8a80;
  text-shadow: #000 0 0 2px;
}
</style>
