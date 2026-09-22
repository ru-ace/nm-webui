import { writable } from 'svelte/store';
import type { ComponentType } from 'svelte';

export interface HeaderAction {
  // Used for the icon-only mobile header button (aria-label / tooltip).
  label: string;
  icon: ComponentType;
  onClick: () => void;
  disabled?: boolean;
  // Render a spinner instead of the icon while loading.
  loading?: boolean;
}

// The active page registers its primary action button here so the mobile
// navbar can show it as an icon-only button in the header.
export const headerAction = writable<HeaderAction | null>(null);