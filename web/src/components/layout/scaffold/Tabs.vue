<template>
  <div class="flex flex-wrap mt-2 md:gap-4">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      class="border-transparent w-full py-1 md:w-auto flex cursor-pointer md:border-b-2 text-wp-text-100 items-center"
      :class="{
        'border-wp-text-100': activeTab === tab.id,
        'border-transparent': activeTab !== tab.id,
      }"
      type="button"
      @click="selectTab(tab)"
    >
      <Icon v-if="activeTab === tab.id" name="chevron-right" class="md:hidden flex-shrink-0" />
      <Icon v-else name="blank" class="md:hidden" />
      <span class="flex gap-2 items-center md:justify-center flex-row py-1 px-2 w-full min-w-20 dark:hover:bg-wp-background-100 hover:bg-wp-background-200 rounded-md">
        <Icon v-if="tab.icon" :name="tab.icon" :class="tab.iconClass" class="flex-shrink-0" />
        <span>{{ tab.title }}</span>
        <CountBadge v-if="tab.count" :value="tab.count" />
      </span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router';

import CountBadge from '~/components/atomic/CountBadge.vue';
import Icon from '~/components/atomic/Icon.vue';
import { useTabsClient, type Tab } from '~/compositions/useTabs';

const router = useRouter();
const route = useRoute();

const { activeTab, tabs, disableUrlHashMode } = useTabsClient();

async function selectTab(tab: Tab) {
  if (tab.id === undefined) {
    return;
  }

  activeTab.value = tab.id;

  if (!disableUrlHashMode.value) {
    await router.replace({ params: route.params, hash: `#${tab.id}` });
  }
}
</script>
