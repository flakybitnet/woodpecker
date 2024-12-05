import { useI18n } from 'vue-i18n';
import '@formatjs/intl-durationformat/polyfill';

let currentLocale: string;
let relativeFormatter: Intl.RelativeTimeFormat;
let durationFormatter: Intl.DurationFormat;

setDateLocale('en')

function splitDuration(durationMs: number) { // for example 3222111444
  const totalSeconds = durationMs / 1000; // for example 3222111.444
  const totalMinutes = totalSeconds / 60; // for example 53701.8574
  const totalHours = totalMinutes / 60; // for example 895.03095
  const totalDays = totalHours / 24; // for example 37.29295

  const seconds = Math.floor(totalSeconds) % 60; // for 3222111.444 total seconds it will be 51
  const minutes = Math.floor(totalMinutes) % 60; // for 53701.8574 total minutes it will be 1
  const hours = Math.floor(totalHours) % 24; // for 895.03095 total hours it will be 7
  const days = Math.floor(totalDays); // for 37.29295 total days it will be 37

  return {
    seconds,
    minutes,
    hours,
    days,
    totalHours,
    totalMinutes,
    totalSeconds,
    totalDays
  };
}

function toLocaleString(date: Date) {
  return date.toLocaleString(currentLocale, {
    dateStyle: 'short',
    timeStyle: 'short',
  });
}

function timeAgo(date: number) {
  const d = splitDuration(Date.now() - date);

  if (d.totalDays > 365) {
    return relativeFormatter.format(-Math.floor(d.totalDays/365), 'year');
  }
  if (d.totalDays > 30) {
    return relativeFormatter.format(-Math.floor(d.totalDays/30), 'month');
  }
  if (d.totalDays > 1) {
    return relativeFormatter.format(-Math.floor(d.totalDays), 'day');
  }
  if (d.totalHours > 1) {
    return relativeFormatter.format(-Math.floor(d.totalHours), 'hour');
  }
  if (d.totalMinutes > 1) {
    return relativeFormatter.format(-Math.floor(d.totalMinutes), 'minute');
  }
  return useI18n().t('time.recently');
}

function prettyDuration(durationMs: number) {
  const d = splitDuration(durationMs);
  return durationFormatter.format({days: d.days, hours: d.hours, minutes: d.minutes, seconds: d.seconds});
}

function durationAsNumber(durationMs: number): string {
  const { seconds, minutes, hours } = splitDuration(durationMs);

  // 1:1 => 01:01
  const minutesSecondsFormatted = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;

  if (hours > 0) {
    return `${hours.toString().padStart(2, '0')}:${minutesSecondsFormatted}`; // 1:1:1 => 01:01:01
  }

  return minutesSecondsFormatted;
}

async function setDateLocale(locale: string) {
  currentLocale = locale;
  relativeFormatter = new Intl.RelativeTimeFormat(currentLocale, { style: "narrow" });
  durationFormatter = new Intl.DurationFormat(currentLocale, { style: "narrow" });
}

export function useDate() {
  return {
    toLocaleString,
    timeAgo,
    prettyDuration,
    setDateLocale,
    durationAsNumber,
  };
}
