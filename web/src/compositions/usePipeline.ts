import { computed, type Ref } from 'vue';

import { useDate } from '~/compositions/useDate';
import { useElapsedTime } from '~/compositions/useElapsedTime';
import type { Pipeline } from '~/lib/api/types';
import { convertEmojis } from '~/utils/emoji';

const { toLocaleString, timeAgo, prettyDuration } = useDate();

export default (pipeline: Ref<Pipeline | undefined>) => {
  const sinceRaw = computed(() => {
    if (!pipeline.value) {
      return undefined;
    }

    const start = pipeline.value.created_at || 0;

    return start * 1000;
  });

  const sinceUnderOneHour = computed(
    () => sinceRaw.value !== undefined && sinceRaw.value > 0 && sinceRaw.value <= 1000 * 60 * 60,
  );
  const { time: sinceElapsed } = useElapsedTime(sinceUnderOneHour, sinceRaw);

  const since = computed(() => {
    if (sinceRaw.value === 0 || sinceElapsed.value === undefined) {
      return '—';
    }

    // TODO: check whether elapsed works
    return timeAgo(sinceElapsed.value);
  });

  const running = computed(() => pipeline.value !== undefined && pipeline.value.status === 'running');

  const durationRaw = computed(() => {
    if (!pipeline.value) {
      return undefined;
    }

    const start = pipeline.value.started_at || 0;
    const end = pipeline.value.finished_at || pipeline.value.updated_at || 0;

    if (start === 0 || end === 0) {
      return 0;
    }

    let duration = end - start; // in seconds since 1970
    if (running.value) {
      duration = Date.now() / 1000 - start; // only calculate time based no now() for running pipelines
    }

    return duration * 1000; // in milliseconds
  });

  const { time: durationElapsed } = useElapsedTime(running, durationRaw);

  const duration = computed(() => {
    if (durationRaw.value === 0 || durationElapsed.value === undefined) {
      return '—';
    }

    return prettyDuration(durationElapsed.value);
  });

  const message = computed(() => convertEmojis(pipeline.value?.message ?? ''));
  const shortMessage = computed(() => message.value.split('\n')[0]);

  const prTitleWithDescription = computed(() => convertEmojis(pipeline.value?.title ?? ''));
  const prTitle = computed(() => prTitleWithDescription.value.split('\n')[0]);

  const prettyRef = computed(() => {
    if (pipeline.value?.event === 'tag' || pipeline.value?.event === 'release') {
      return pipeline.value.ref.replaceAll('refs/tags/', '');
    }

    return pipeline.value?.branch;
  });

  const created = computed(() => {
    if (!pipeline.value) {
      return undefined;
    }

    const start = pipeline.value.created_at || 0;

    return toLocaleString(new Date(start * 1000));
  });

  return { since, duration, message, shortMessage, prTitle, prTitleWithDescription, prettyRef, created };
};
