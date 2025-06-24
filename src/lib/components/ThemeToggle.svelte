<script lang="ts">
import { Button, Icon, ThemeToggler } from '@sveltestrap/sveltestrap';
import { m } from '$lib/paraglide/messages.js';
</script>

<ThemeToggler let:currentColorMode let:toggleColorMode>
<Button outline aria-label={m.toggle_theme()} on:click={() => toggleColorMode()}>
    <Icon name={currentColorMode === 'light' ? 'moon' : 'sun'} />
</Button> 
</ThemeToggler>