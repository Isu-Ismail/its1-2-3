import { writable } from 'svelte/store';

export type ActiveTab = 'dashboard' | 'download' | 'config';

const getInitialTab = (): ActiveTab => {
  if (typeof window === 'undefined') return 'dashboard';
  const saved = sessionStorage.getItem('ctrlv_active_tab') as ActiveTab;
  if (saved === 'history' as any) return 'dashboard';
  return saved || 'dashboard';
};

export const activeTabStore = writable<ActiveTab>(getInitialTab());

activeTabStore.subscribe((tab) => {
  if (typeof window !== 'undefined') {
    sessionStorage.setItem('ctrlv_active_tab', tab);
  }
});

export function setActiveTab(tab: ActiveTab) {
  activeTabStore.set(tab);
}
