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

    <div
      v-if="visibleSequenceGroup"
      class="sequence-group"
      :title="sequenceTitle(visibleSequenceGroup)"
    >
      <span class="sequence-label">
        {{ sequenceKindLabel(visibleSequenceGroup.kind) }} {{ visibleSequenceGroup.items?.length || 0 }}
      </span>
      <span v-if="sequenceStateLabel" class="sequence-state">
        {{ sequenceStateLabel }}
      </span>
      <span v-if="currentFaceEvidence" class="face-evidence">
        {{ faceEvidenceLabel(currentFaceEvidence) }}
      </span>
    </div>

    <div
      v-for="group in visibleDuplicateGroups"
      :key="group.group_id"
      class="duplicate-group"
      :title="groupTitle(group)"
    >
      <span class="duplicate-label">
        {{ group.kind === 'exact' ? '精确重复' : '相似' }} {{ group.items.length }}
      </span>
      <ui-icon
        light
        size="24"
        class="duplicate-action not-duplicate"
        :class="{ active: statusAction === 'NOT_DUPLICATE', busy }"
        title="不是重复"
        @click.stop="submitDuplicate('NOT_DUPLICATE', group)"
      >
        link_off
      </ui-icon>
      <ui-icon
        light
        size="24"
        class="duplicate-action confirm-duplicate"
        :class="{ active: statusAction === 'DUPLICATE_CONFIRMED', busy }"
        title="确认重复（只标记待删，仍需后续审批）"
        @click.stop="submitDuplicate('DUPLICATE_CONFIRMED', group)"
      >
        content_copy
      </ui-icon>
    </div>

    <ui-icon
      v-if="sequenceError && !error"
      light
      class="duplicate-warning"
      size="20"
      :title="sequenceError"
    >
      warning
    </ui-icon>
    <ui-icon
      v-if="duplicateError && !error"
      light
      class="duplicate-warning"
      size="20"
      :title="duplicateError"
    >
      warning
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
const duplicateGroups = ref([]);
const sequenceGroups = ref([]);
const busy = ref(false);
const connected = ref(false);
const error = ref('');
const duplicateError = ref('');
const sequenceError = ref('');
let requestVersion = 0;

const visibleDuplicateGroups = computed(() => {
  const exact = duplicateGroups.value.filter(group => group.kind === 'exact');
  return exact.length ? exact : duplicateGroups.value.filter(group => group.kind === 'similar');
});

const visibleSequenceGroup = computed(() => sequenceGroups.value[0] || null);

const currentSequenceItem = computed(() => {
  const group = visibleSequenceGroup.value;
  const id = Number(fileId.value);
  if (!group || !Number.isInteger(id)) return null;
  return (group.items || []).find(item => Number(item.file_id) === id) || null;
});

const currentFaceEvidence = computed(() => currentSequenceItem.value?.face_evidence || null);

const sequenceStateLabel = computed(() => {
  const group = visibleSequenceGroup.value;
  if (!group) return '';
  if (group.protected) return '保护序列';
  const recommended = Number(group.recommended_keeper_file_id);
  const current = Number(fileId.value);
  if (Number.isInteger(recommended) && recommended > 0) {
    if (recommended === current) return '建议精选';
    const item = (group.items || []).find(row => Number(row.file_id) === recommended);
    return `建议查看 ${item?.name || ('#' + recommended)}`;
  }
  return '人工复核';
});

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

function sequenceKindLabel(kind) {
  return ({
    burst: '连拍',
    exposure_bracket: '包围曝光',
    panorama: '全景序列',
    hdr_panorama: 'HDR 全景',
    unknown: '序列待确认',
  })[kind] || '序列';
}

function faceEvidenceLabel(evidence) {
  const count = Number(evidence?.face_count);
  if (!Number.isInteger(count) || count < 0) return '';
  if (count === 0) return '未检出人脸';
  const parts = [`人脸 ${count}`];
  const quality = Number(evidence?.mean_capture_quality);
  const eye = Number(evidence?.min_eye_aspect);
  if (Number.isFinite(quality)) parts.push(`人像质 ${quality.toFixed(2)}`);
  if (Number.isFinite(eye)) parts.push(`眼部 ${eye.toFixed(2)}`);
  return parts.join(' · ');
}

function sequenceTitle(group) {
  const parts = [`${sequenceKindLabel(group.kind)} · ${group.items?.length || 0} 张`];
  if (group.protected) parts.push('仅保护/复核，不参与最佳帧推荐');
  else if (group.recommended_keeper_file_id) parts.push('推荐仅供参考，不改变保留/删除状态');
  if (currentFaceEvidence.value) parts.push(faceEvidenceLabel(currentFaceEvidence.value));
  return parts.join('；');
}

function groupTitle(group) {
  const kind = group.kind === 'exact' ? '精确重复' : '视觉相似';
  const members = (group.items || []).map(item => {
    const diff = item.difference == null ? '' : ` (差异 ${item.difference})`;
    return `${item.name || ('#' + item.file_id)}${diff}`;
  });
  return `${kind}：${members.join(' · ')}`;
}

async function loadDuplicateGroups(collection, id, version) {
  try {
    const query = new URLSearchParams({
      collection_id: collection,
      file_id: String(id),
    });
    const response = await fetch(`${sidecarUrl.value}/api/culling/duplicate-groups?${query}`, {
      method: 'GET',
      cache: 'no-store',
      credentials: 'omit',
    });
    const payload = await parseResponse(response);
    if (version !== requestVersion) return;
    duplicateGroups.value = Array.isArray(payload?.groups) ? payload.groups : [];
    duplicateError.value = '';
  } catch (e) {
    if (version !== requestVersion) return;
    duplicateGroups.value = [];
    duplicateError.value = `重复候选不可用：${e?.message || e}`;
  }
}

async function loadSequenceGroups(collection, id, version) {
  try {
    const query = new URLSearchParams({
      collection_id: collection,
      file_id: String(id),
    });
    const response = await fetch(`${sidecarUrl.value}/api/culling/sequence-groups?${query}`, {
      method: 'GET',
      cache: 'no-store',
      credentials: 'omit',
    });
    const payload = await parseResponse(response);
    if (version !== requestVersion) return;
    sequenceGroups.value = Array.isArray(payload?.groups) ? payload.groups : [];
    sequenceError.value = '';
  } catch (e) {
    if (version !== requestVersion) return;
    sequenceGroups.value = [];
    sequenceError.value = `序列证据不可用：${e?.message || e}`;
  }
}

async function loadStatus() {
  const collection = collectionId.value;
  const id = Number(fileId.value);
  if (!collection || !Number.isInteger(id) || id <= 0) {
    statusAction.value = null;
    duplicateGroups.value = [];
    sequenceGroups.value = [];
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
  await Promise.all([
    loadDuplicateGroups(collection, id, version),
    loadSequenceGroups(collection, id, version),
  ]);
}

async function submit(action, extra = {}) {
  if (busy.value) return false;
  const collection = collectionId.value;
  const id = Number(fileId.value);
  if (!collection || !Number.isInteger(id) || id <= 0) return false;
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
        ...extra,
      }),
    });
    const payload = await parseResponse(response);
    statusAction.value = payload?.status?.action || action;
    connected.value = true;
    error.value = '';
    return true;
  } catch (e) {
    connected.value = false;
    error.value = `选片记录失败：${e?.message || e}`;
    return false;
  } finally {
    busy.value = false;
  }
}

async function submitDuplicate(action, group) {
  if (!group?.group_id) return;
  if (action === 'DUPLICATE_CONFIRMED') {
    const label = group.kind === 'exact' ? '精确重复' : '视觉相似';
    const ok = window.confirm(
      `确认把当前照片标记为“${label}待删候选”？\n\n这一步不会删除文件，后续仍需 delete-plan、人工审批和隔离流程。`
    );
    if (!ok) return;
  }
  await submit(action, { group_id: group.group_id });
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
  gap: 2px;
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

.culling-icon.busy,
.duplicate-action.busy {
  opacity: 0.45;
  pointer-events: none;
}

.sequence-group {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 4px 0 8px;
  padding: 6px 10px;
  border: 1px solid rgba(144, 202, 249, 0.5);
  border-radius: 999px;
  background: rgba(20, 20, 20, 0.68);
  color: white;
  backdrop-filter: blur(7px);
}

.sequence-label,
.sequence-state,
.face-evidence {
  font-size: 11px;
  line-height: 1;
  white-space: nowrap;
}

.sequence-label {
  color: rgba(187, 222, 251, 0.98);
}

.sequence-state {
  color: rgba(255, 224, 130, 0.98);
}

.face-evidence {
  color: rgba(200, 230, 201, 0.98);
}

.duplicate-group {
  display: flex;
  align-items: center;
  gap: 3px;
  margin: 0 4px 0 8px;
  padding: 3px 5px 3px 9px;
  border: 1px solid rgba(255, 196, 77, 0.48);
  border-radius: 999px;
  background: rgba(20, 20, 20, 0.62);
  color: white;
  backdrop-filter: blur(7px);
}

.duplicate-label {
  font-size: 11px;
  line-height: 1;
  white-space: nowrap;
  color: rgba(255, 225, 164, 0.95);
}

.duplicate-action {
  padding: 7px 5px;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.82);
  text-shadow: #000 0 0 2px;
}

.duplicate-action:hover {
  color: white;
}

.duplicate-action.not-duplicate.active {
  color: #8ee6a5;
}

.duplicate-action.confirm-duplicate.active {
  color: #ff9e93;
}

.culling-actions.disconnected .culling-icon:not(.active) {
  opacity: 0.55;
}

.duplicate-warning {
  padding: 20px 5px 20px 4px;
  color: #ffd180;
  text-shadow: #000 0 0 2px;
}

.culling-error {
  padding: 20px 10px 20px 4px;
  color: #ff8a80;
  text-shadow: #000 0 0 2px;
}
</style>
