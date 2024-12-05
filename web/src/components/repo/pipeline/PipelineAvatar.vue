<template>
  <SvgIcon v-if="pipeline.event === 'cron'" :path="mdiAlarm" :size="size || defaultSize" title="Cron" />
  <img v-else class="rounded-md" :class="`w-${Math.floor(+(size || defaultSize) / 4)}`"
       :src="pipeline.author_avatar" :title="pipeline.author" :alt="`${pipeline.author} avatar`"
  />
</template>

<script lang="ts" setup>
import { mdiAlarm } from '@mdi/js';

import SvgIcon from '../../atomic/SvgIcon.vue';
import type { Pipeline } from '~/lib/api/types';


defineProps<{
  pipeline: Pipeline;
  size?: number | string;
}>();

const defaultSize = 32;

</script>
