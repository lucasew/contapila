<script lang="ts">
import { Badge } from '@sveltestrap/sveltestrap';
export let tipo: string;

const tipoToColor: Record<string, string> = {
  transaction: 'primary',
  open: 'success',
  close: 'danger',
  balance: 'info',
  price: 'warning',
  note: 'secondary'
};

const color = tipoToColor[tipo] || 'secondary';
</script>

<Badge color={color} outline>{tipo?.toUpperCase()}</Badge> 