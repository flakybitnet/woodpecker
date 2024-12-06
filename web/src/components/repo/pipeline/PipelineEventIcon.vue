<template>
  <Icon v-if="pipeline.event === 'pull_request'" name="pull-request" class="flex-shrink-0" />
  <Icon v-else-if="pipeline.event === 'pull_request_closed'" name="pull-request-closed" class="flex-shrink-0" />
  <Icon v-else-if="pipeline.event === 'deployment'" name="deployment" class="flex-shrink-0" />
  <Icon v-else-if="pipeline.event === 'tag' || pipeline.event === 'release'" name="tag" class="flex-shrink-0" />
  <Icon v-else-if="pipeline.event === 'cron'" name="stopwatch" class="flex-shrink-0" />
  <Icon v-else-if="pipeline.event === 'manual'" name="manual-pipeline" class="flex-shrink-0" />
  <Icon v-else name="push" class="flex-shrink-0" />
</template>

<script lang="ts" setup>
import type { Pipeline } from '~/lib/api/types';
import Icon from '~/components/atomic/Icon.vue';


defineProps<{
  pipeline: Pipeline;
}>();

</script>
