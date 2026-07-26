const RTF = new Intl.RelativeTimeFormat('ja', { numeric: 'auto' });

const UNITS: { unit: Intl.RelativeTimeFormatUnit; seconds: number }[] = [
  { unit: 'year', seconds: 31536000 },
  { unit: 'month', seconds: 2592000 },
  { unit: 'day', seconds: 86400 },
  { unit: 'hour', seconds: 3600 },
  { unit: 'minute', seconds: 60 },
];

export const formatRelativeTime = (isoDate: string): string => {
  const date = new Date(isoDate);
  const diffSeconds = (date.getTime() - Date.now()) / 1000;

  for (const { unit, seconds } of UNITS) {
    if (Math.abs(diffSeconds) >= seconds) {
      return RTF.format(Math.round(diffSeconds / seconds), unit);
    }
  }
  return RTF.format(Math.round(diffSeconds), 'second');
};
