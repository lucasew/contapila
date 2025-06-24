<script lang="ts">
import { Dropdown, DropdownToggle, DropdownMenu, DropdownItem } from '@sveltestrap/sveltestrap';
import { setLocale, getLocale } from '$lib/paraglide/runtime.js';
import { onMount } from 'svelte';

let dropdownOpen = false;
let currentLocale = '';
const languages = [
  { code: 'pt-br', emoji: '🇧🇷', label: 'Português' },
  { code: 'en', emoji: '🇺🇸', label: 'English' }
];

onMount(() => {
  currentLocale = getLocale();
});

function handleLocaleChange(locale: string) {
  setLocale(locale);
  currentLocale = locale;
  dropdownOpen = false;
}
</script>

<Dropdown isOpen={dropdownOpen} toggle={() => dropdownOpen = !dropdownOpen}>
  <DropdownToggle caret class="btnl">
    {languages.find(l => l.code === currentLocale)?.emoji || currentLocale}
  </DropdownToggle>
  <DropdownMenu>
    {#each languages as lang}
      <DropdownItem on:click={() => handleLocaleChange(lang.code)} active={currentLocale === lang.code}>
        {lang.emoji} {lang.label}
      </DropdownItem>
    {/each}
  </DropdownMenu>
</Dropdown> 