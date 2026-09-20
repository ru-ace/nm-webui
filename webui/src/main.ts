import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';

const storedTheme = window.localStorage.getItem('nm-webui-theme');
const initialTheme = storedTheme === 'light' || storedTheme === 'dark'
  ? storedTheme
  : window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
document.documentElement.dataset.theme = initialTheme;

const app = mount(App, {
  target: document.getElementById('app')!,
});

export default app;
