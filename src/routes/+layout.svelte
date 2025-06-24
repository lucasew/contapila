<script lang="ts">
import { Navbar, NavbarBrand, Container, Styles, Progress, ThemeToggler, Dropdown, DropdownToggle, DropdownMenu, DropdownItem, Nav, NavItem, ButtonGroup } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/components/ThemeToggle.svelte';
import { navigating } from '$app/stores';
import { writable } from 'svelte/store';
import { setContext } from 'svelte';
import { m } from '$lib/paraglide/messages.js';
import { setLocale, getLocale } from '$lib/paraglide/runtime.js';
import LanguageSelector from '$lib/components/LanguageSelector.svelte';

// Store local para processamento com contador
const processingStore = writable({ activeTasks: 0 });

// Disponibiliza o store globalmente
setContext('processingStore', processingStore);

$: showProgress = $navigating || $processingStore.activeTasks > 0;

let dropdownOpen = false;
let currentLocale = getLocale();
const languages = [
  { code: 'pt-br', emoji: '🇧🇷', label: 'Português' },
  { code: 'en', emoji: '🇺🇸', label: 'English' }
];

function handleLocaleChange(locale) {
  setLocale(locale);
  currentLocale = locale;
  dropdownOpen = false;
}
</script>

<ThemeToggler let:currentColorMode>
	<Progress 
		striped={showProgress}
		animated={showProgress} 
		value={100} 
		color={currentColorMode === 'light' ? 'dark' : 'light'}
		class="rounded-0"
	/>
</ThemeToggler>

<Styles/>
<Navbar>
    <NavbarBrand href="/">{m.app_name() }</NavbarBrand>
    <div class="d-flex align-items-center ms-auto gap-2">
      <ThemeToggle />
      <LanguageSelector />
    </div>
</Navbar>
<Container>
  <slot />
</Container>
