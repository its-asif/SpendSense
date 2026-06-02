import timezoneData from '../data/timezones.json';

type TimezoneEntry = {
  zone: string;
  gmt: string;
  name: string;
};

export type TimezoneOption = TimezoneEntry & {
  label: string;
};

function decodeEntities(value: string) {
  return value.replace(/&amp;/g, '&');
}

const timezones = timezoneData as TimezoneEntry[];

export const timezoneOptions: TimezoneOption[] = timezones.map((entry) => {
  const name = decodeEntities(entry.name);
  return {
    ...entry,
    name,
    label: `${entry.gmt} ${name} (${entry.zone})`,
  };
});
