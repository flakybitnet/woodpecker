<template>
  <div
    class="flex gap-2 items-center text-gray-700 font-bold rounded-md p-2 border border-solid border-l-6 border-wp-hint-warn-200 bg-wp-hint-warn-100"
  >
    <Icon v-if="!textOnly" name="warning" />
    <slot>
      <span class="whitespace-pre">{{ text }}</span>
    </slot>
  </div>
</template>

<script lang="ts" setup>
import Icon from './Icon.vue';

defineProps<{
  textOnly?: boolean;
  text?: string;
}>();
</script>
