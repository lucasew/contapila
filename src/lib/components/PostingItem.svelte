<script lang="ts">
import { Row, Col, Collapse, Icon, Accordion, AccordionItem } from '@sveltestrap/sveltestrap';
const { posting, open = false } = $props();
let isOpen = $state(open);
</script>
<Accordion>
    <AccordionItem>
        <Row slot="header" class='w-100'>
            <Col>{posting.account}</Col> 
            <Col class='text-end'>
              {posting.amount?.value ?? ''} {posting.amount?.currency ?? ''}
            </Col>
          </Row>


    <div class="ms-4">
      {#if posting.meta}
        <ul class="ms-3 mb-2">
          {#each Object.entries(posting.meta) as [k, v]}
            <li><strong>{k}:</strong> <span class="ms-3">{JSON.stringify(v)}</span></li>
          {/each}
        </ul>
      {:else}
        <span class="ms-3 text-muted">Sem atributos extras</span>
      {/if}
    </div>
    </AccordionItem>

</Accordion>