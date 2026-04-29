import dayjs from 'dayjs';

export function toRFC3339(d: Date | string | number,): string {
  return dayjs(d,).format('YYYY-MM-DDTHH:mm:ssZ',);
}

export function startOfDay(d: Date | string | number,): Date {
  return dayjs(d,).startOf('day',).toDate();
}

export function endOfDay(d: Date | string | number,): Date {
  return dayjs(d,).endOf('day',).toDate();
}

export function addDays(d: Date | string | number, n: number,): Date {
  return dayjs(d,).add(n, 'day',).toDate();
}

export function toDateKey(d: Date | string | number,): string {
  return dayjs(d,).format('YYYY-MM-DD',);
}

export function formatDateTime(d: Date | string | number,): string {
  return dayjs(d,).format('YYYY-MM-DD HH:mm:ss',);
}

export function formatDate(d: Date | string | number,): string {
  return dayjs(d,).format('YYYY-MM-DD',);
}

export function formatTime(d: Date | string | number,): string {
  return dayjs(d,).format('HH:mm:ss',);
}

export function formatShortDate(d: Date | string | number,): string {
  return dayjs(d,).format('M/D',);
}

export function isValidDate(d: Date | string | number,): boolean {
  return dayjs(d,).isValid();
}

export function now(): Date {
  return dayjs().toDate();
}
