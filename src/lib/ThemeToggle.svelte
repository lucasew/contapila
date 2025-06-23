<script lang="ts">
import { Button, Icon, ThemeToggler } from '@sveltestrap/sveltestrap';


</script>

<ThemeToggler let:currentColorMode let:toggleColorMode>
<Button outline aria-label="Alternar tema" on:click={() => toggleColorMode()}>
    <Icon name={currentColorMode === 'light' ? 'moon' : 'sun'} />
</Button> 
</ThemeToggler>