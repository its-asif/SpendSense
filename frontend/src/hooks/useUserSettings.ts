import { useEffect, useState } from 'react';
import { readUserSettings, subscribeToUserSettings, type UserSettings } from '../lib/userSettings';

export function useUserSettings() {
  const [settings, setSettings] = useState<UserSettings>(() => readUserSettings());

  useEffect(() => {
    return subscribeToUserSettings(() => {
      setSettings(readUserSettings());
    });
  }, []);

  return settings;
}
