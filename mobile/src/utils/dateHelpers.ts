import { format, formatDistanceToNow, isPast, parseISO } from 'date-fns';
import { ru } from 'date-fns/locale';

export function formatWorkOrderDate(dateStr: string): string {
  return format(parseISO(dateStr), 'dd MMM, HH:mm', { locale: ru });
}

export function formatSLADeadline(dateStr: string): string {
  const date = parseISO(dateStr);
  return format(date, 'dd MMM, HH:mm', { locale: ru });
}

export function isSLAPast(dateStr: string): boolean {
  return isPast(parseISO(dateStr));
}

export function formatRelativeTime(dateStr: string): string {
  return formatDistanceToNow(parseISO(dateStr), { addSuffix: true, locale: ru });
}

export function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 6) return 'Доброй ночи';
  if (hour < 12) return 'Доброе утро';
  if (hour < 18) return 'Добрый день';
  return 'Добрый вечер';
}