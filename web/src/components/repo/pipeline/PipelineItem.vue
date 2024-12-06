<template>
  <ListItem v-if="pipeline" class="p-0 w-full">
    <div class="flex w-11 items-center md:mr-4">
      <div
        class="h-full w-3"
        :class="{
          'bg-wp-state-warn-100': pipeline.status === 'pending',
          'bg-wp-state-error-100': pipelineStatusColors[pipeline.status] === 'red',
          'bg-wp-state-neutral-100': pipelineStatusColors[pipeline.status] === 'gray',
          'bg-wp-state-ok-100': pipelineStatusColors[pipeline.status] === 'green',
          'bg-wp-state-info-100': pipelineStatusColors[pipeline.status] === 'blue',
        }"
        :title="`Pipeline: ${pipeline.number}`"
      />
      <div class="w-8 flex flex-wrap justify-between items-center h-full">
        <PipelineRunningIcon v-if="pipeline.status === 'started' || pipeline.status === 'running'" />
        <PipelineStatusIcon v-else class="mx-2 md:mx-3" :status="pipeline.status" size="28" />
      </div>
    </div>

    <div class="flex py-2 px-4 flex-grow min-w-0 <md:flex-wrap text-wp-text-100">
      <div class="w-full md:w-auto md:mr-4 flex items-center space-x-2 min-w-0" :title="`Event: ${pipelineEventTitle}`">
        <PipelineEventIcon :pipeline="pipeline" />
        <span class="<md:underline whitespace-nowrap overflow-hidden overflow-ellipsis" :title="message">
          {{ shortMessage }}
        </span>
      </div>

      <div class="grid grid-rows-2 grid-cols-2 grid-flow-col w-full md:ml-auto md:w-96 py-2 gap-x-4 gap-y-2 flex-shrink-0 md:mr-4 ">
        <div class="flex space-x-2 items-center min-w-0" :title="`${(pipeline.event === 'tag' || pipeline.event === 'release') ? 'Tag' : 'Branch'}: ${prettyRef}`">
          <Icon v-if="pipeline.event === 'tag' || pipeline.event === 'release'" name="tag" class="flex-shrink-0" />
          <Icon v-else name="branch" class="flex-shrink-0" />
          <span class="truncate">{{ prettyRef }}</span>
        </div>

        <div class="flex space-x-2 items-center min-w-0" :title="`Head commit: ${pipeline.commit}`">
          <Icon name="commit" class="flex-shrink-0" />
          <span class="truncate">{{ pipeline.commit.slice(0, 4).concat(' ', pipeline.commit.slice(4, 8)) }}</span>
        </div>

        <div class="flex space-x-2 items-center min-w-0" :title="i18n.t('repo.pipeline.duration')">
          <Icon name="duration" class="flex-shrink-0" />
          <span class="truncate">{{ duration }}</span>
        </div>

        <div class="flex space-x-2 items-center min-w-0" :title="i18n.t('repo.pipeline.created', { created })">
          <Icon name="since"  class="flex-shrink-0" />
          <span class="truncate">{{ since }}</span>
        </div>
      </div>

      <div class="<md:hidden flex items-center flex-shrink-0 mx-2">
        <PipelineAvatar :pipeline />
      </div>
    </div>
  </ListItem>
</template>

<style scoped>
.grid-cols-2 {
  grid-template-columns: 2fr 1fr;
}
</style>


<script lang="ts" setup>
import { computed, toRef } from 'vue';
import { useI18n } from 'vue-i18n';

import Icon from '~/components/atomic/Icon.vue';
import ListItem from '~/components/atomic/ListItem.vue';
import { pipelineStatusColors } from '~/components/repo/pipeline/pipeline-status';
import PipelineRunningIcon from '~/components/repo/pipeline/PipelineRunningIcon.vue';
import PipelineStatusIcon from '~/components/repo/pipeline/PipelineStatusIcon.vue';
import usePipeline from '~/compositions/usePipeline';
import type { Pipeline } from '~/lib/api/types';
import PipelineAvatar from '~/components/repo/pipeline/PipelineAvatar.vue';
import PipelineEventIcon from '~/components/repo/pipeline/PipelineEventIcon.vue';

const props = defineProps<{
  pipeline: Pipeline;
}>();

const i18n = useI18n();

const pipeline = toRef(props, 'pipeline');
const { since, duration, message, shortMessage, prettyRef, created } = usePipeline(pipeline);

const pipelineEventTitle = computed(() => {
  switch (pipeline.value.event) {
    case 'pull_request':
      return i18n.t('repo.pipeline.event.pr');
    case 'pull_request_closed':
      return i18n.t('repo.pipeline.event.pr_closed');
    case 'deployment':
      return i18n.t('repo.pipeline.event.deploy');
    case 'tag':
      return i18n.t('repo.pipeline.event.tag');
    case 'release':
      return i18n.t('repo.pipeline.event.release');
    case 'cron':
      return i18n.t('repo.pipeline.event.cron');
    case 'manual':
      return i18n.t('repo.pipeline.event.manual');
    default:
      return i18n.t('repo.pipeline.event.push');
  }
});
</script>
