import { useEffect, useState } from 'react';

const SIDEBAR_STATE_KEY = 'spendsense-sidebar-collapsed';
const SIDEBAR_STATE_EVENT = 'spendsense-sidebar-collapsed-change';

function readCollapsedState() {
  if (typeof window === 'undefined') {
    return false;
  }

  return window.localStorage.getItem(SIDEBAR_STATE_KEY) === '1';
}

export function useSidebarCollapsed() {
  const [collapsed, setCollapsed] = useState(() => readCollapsedState());

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const readState = () => readCollapsedState();

    const handleStateChange = () => {
      setCollapsed(readState());
    };

    window.addEventListener('storage', handleStateChange);
    window.addEventListener(SIDEBAR_STATE_EVENT, handleStateChange);

    return () => {
      window.removeEventListener('storage', handleStateChange);
      window.removeEventListener(SIDEBAR_STATE_EVENT, handleStateChange);
    };
  }, []);

  const updateCollapsed = (next: boolean) => {
    if (typeof window === 'undefined') {
      setCollapsed(next);
      return;
    }

    window.localStorage.setItem(SIDEBAR_STATE_KEY, next ? '1' : '0');
    window.dispatchEvent(new Event(SIDEBAR_STATE_EVENT));
    setCollapsed(next);
  };

  const toggleCollapsed = () => {
    updateCollapsed(!collapsed);
  };

  return { collapsed, setCollapsed: updateCollapsed, toggleCollapsed };
}
