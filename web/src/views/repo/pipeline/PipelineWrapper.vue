<template>
  <Scaffold
    v-if="pipeline && repo"
    v-model:active-tab="activeTab"
    enable-tabs
    disable-tab-url-hash-mode
    :go-back="goBack"
    :fluid-content="activeTab === 'tasks'"
    full-width-header
  >
    <template #title>
      <span>
        <router-link :to="{ name: 'org', params: { orgId: repo.org_id } }" class="hover:underline">
          {{ repo.owner }}
        </router-link> /
        <router-link :to="{ name: 'repo' }" class="hover:underline">{{ repo.name }}</router-link> /
        {{ pipelineId }}
      </span>
    </template>

    <template #titleMiddle>
      <div class="flex items-center space-x-2 min-w-0">
        <PipelineEventIcon :pipeline="pipeline" size="22" />
        <span class="text-xl font-semibold min-w-0 whitespace-nowrap overflow-hidden overflow-ellipsis" :title="message">
          {{ shortMessage }}
        </span>
      </div>
    </template>

    <template #titleActions>
      <span /> <!-- durations block (tabActions slot) depends on this one (titleActions), see Header.vue:50 -->
      <!-- so, if guest browses public repo, he won't see not only pipeline actions (restart, etc.), but also durations and user avatar -->

      <template v-if="repoPermissions!.push && pipeline.status !== 'declined' && pipeline.status !== 'blocked'">
        <div class="flex content-start gap-x-2">
          <Button
            v-if="pipeline.status === 'pending' || pipeline.status === 'running'"
            class="flex-shrink-0"
            :text="$t('repo.pipeline.actions.cancel')"
            :is-loading="isCancelingPipeline"
            @click="cancelPipeline"
          />
          <Button
            class="flex-shrink-0"
            :text="$t('repo.pipeline.actions.restart')"
            :is-loading="isRestartingPipeline"
            @click="restartPipeline"
          />
          <Button
            v-if="pipeline.status === 'success' && repo.allow_deploy"
            class="flex-shrink-0"
            :text="$t('repo.pipeline.actions.deploy')"
            @click="showDeployPipelinePopup = true"
          />
          <DeployPipelinePopup
            :pipeline-number="pipelineId"
            :open="showDeployPipelinePopup"
            @close="showDeployPipelinePopup = false"
          />
        </div>
      </template>
    </template>

    <template #tabActions>
      <div class="flex gap-x-4">
        <div class="flex space-x-1 items-center flex-shrink-0" :title="$t('repo.pipeline.duration')">
          <Icon name="duration" />
          <span>{{ duration }}</span>
        </div>
        <div class="flex space-x-1 items-center flex-shrink-0" :title="$t('repo.pipeline.created', { created })">
          <Icon name="since" />
          <span>{{ since }}</span>
        </div>
        <div class="flex space-x-1 items-center flex-shrink-0">
          <PipelineAvatar :pipeline size="20"/>
          <span>{{ pipeline.author }}</span>
        </div>
      </div>
    </template>

    <Tab id="tasks" :title="$t('repo.pipeline.tasks')" />
    <Tab id="config" :title="$t('repo.pipeline.config')" />
    <Tab
      v-if="pipeline.changed_files && pipeline.changed_files.length > 0"
      id="changed-files"
      :title="$t('repo.pipeline.files')"
      :count="pipeline.changed_files?.length"
    />
    <Tab
      v-if="pipeline.errors && pipeline.errors.length > 0"
      id="errors"
      icon="attention"
      :title="pipeline.errors.some((e) => !e.is_warning) ? $t('repo.pipeline.errors') : $t('repo.pipeline.warnings')"
      :count="pipeline.errors?.length"
      :icon-class="pipeline.errors.some((e) => !e.is_warning) ? 'text-wp-state-error-100' : 'text-wp-state-warn-100'"
    />

    <router-view />
  </Scaffold>
</template>

<script lang="ts" setup>
import { computed, inject, onBeforeUnmount, onMounted, provide, ref, toRef, watch, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import Button from '~/components/atomic/Button.vue';
import Icon from '~/components/atomic/Icon.vue';
import DeployPipelinePopup from '~/components/layout/popups/DeployPipelinePopup.vue';
import Scaffold from '~/components/layout/scaffold/Scaffold.vue';
import Tab from '~/components/layout/scaffold/Tab.vue';
import useApiClient from '~/compositions/useApiClient';
import { useAsyncAction } from '~/compositions/useAsyncAction';
import { useFavicon } from '~/compositions/useFavicon';
import useNotifications from '~/compositions/useNotifications';
import usePipeline from '~/compositions/usePipeline';
import { useRouteBack } from '~/compositions/useRouteBack';
import type { PipelineConfig, Repo, RepoPermissions } from '~/lib/api/types';
import { usePipelineStore } from '~/store/pipelines';
import PipelineAvatar from '~/components/repo/pipeline/PipelineAvatar.vue';
import PipelineEventIcon from '~/components/repo/pipeline/PipelineEventIcon.vue';

const props = defineProps<{
  repoId: string;
  pipelineId: string;
}>();

const apiClient = useApiClient();
const route = useRoute();
const router = useRouter();
const notifications = useNotifications();
const favicon = useFavicon();
const i18n = useI18n();

const pipelineStore = usePipelineStore();
const pipelineId = toRef(props, 'pipelineId');
const _repoId = toRef(props, 'repoId');
const repositoryId = computed(() => Number.parseInt(_repoId.value, 10));
const repo = inject<Ref<Repo>>('repo');
const repoPermissions = inject<Ref<RepoPermissions>>('repo-permissions');
if (!repo || !repoPermissions) {
  throw new Error('Unexpected: "repo" & "repoPermissions" should be provided at this place');
}

const pipeline = pipelineStore.getPipeline(repositoryId, pipelineId);
const { since, duration, created, message, shortMessage } = usePipeline(pipeline);
provide('pipeline', pipeline);

const pipelineConfigs = ref<PipelineConfig[]>();
provide('pipeline-configs', pipelineConfigs);

watch(
  pipeline,
  () => {
    favicon.updateStatus(pipeline.value?.status);
  },
  { immediate: true },
);

const showDeployPipelinePopup = ref(false);

async function loadPipeline(): Promise<void> {
  if (!repo) {
    throw new Error('Unexpected: Repo is undefined');
  }

  await pipelineStore.loadPipeline(repo.value.id, Number.parseInt(pipelineId.value, 10));

  if (!pipeline.value?.number) {
    throw new Error('Unexpected: Pipeline number not found');
  }

  pipelineConfigs.value = await apiClient.getPipelineConfig(repo.value.id, pipeline.value.number);
}

const { doSubmit: cancelPipeline, isLoading: isCancelingPipeline } = useAsyncAction(async () => {
  if (!repo) {
    throw new Error('Unexpected: Repo is undefined');
  }

  if (!pipeline.value?.number) {
    throw new Error('Unexpected: Pipeline number not found');
  }

  await apiClient.cancelPipeline(repo.value.id, pipeline.value.number);
  notifications.notify({ title: i18n.t('repo.pipeline.actions.cancel_success'), type: 'success' });
});

const { doSubmit: restartPipeline, isLoading: isRestartingPipeline } = useAsyncAction(async () => {
  if (!repo) {
    throw new Error('Unexpected: Repo is undefined');
  }

  const newPipeline = await apiClient.restartPipeline(repo.value.id, pipelineId.value, {
    fork: true,
  });
  notifications.notify({ title: i18n.t('repo.pipeline.actions.restart_success'), type: 'success' });
  await router.push({
    name: 'repo-pipeline',
    params: { pipelineId: newPipeline.number },
  });
});

onMounted(loadPipeline);
watch([repositoryId, pipelineId], loadPipeline);
onBeforeUnmount(() => {
  favicon.updateStatus('default');
});

const activeTab = computed({
  get() {
    if (route.name === 'repo-pipeline-changed-files') {
      return 'changed-files';
    }

    if (route.name === 'repo-pipeline-config') {
      return 'config';
    }

    if (route.name === 'repo-pipeline-errors') {
      return 'errors';
    }

    return 'tasks';
  },
  set(tab: string) {
    if (tab === 'tasks') {
      router.replace({ name: 'repo-pipeline' });
    }

    if (tab === 'changed-files') {
      router.replace({ name: 'repo-pipeline-changed-files' });
    }

    if (tab === 'config') {
      router.replace({ name: 'repo-pipeline-config' });
    }

    if (tab === 'errors') {
      router.replace({ name: 'repo-pipeline-errors' });
    }
  },
});

const goBack = useRouteBack({ name: 'repo' });
</script>
