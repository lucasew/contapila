<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/beancount.js';
	import { createParser } from '$lib/parser.js';
	import { Table, Badge, Button, Collapse, ListGroup, ListGroupItem } from '@sveltestrap/sveltestrap';

	let files: FileList | undefined = $state();
	let content: any[] = $state([]);
	let erro: string | null = $state(null);
	let openCollapse: Record<number, boolean> = $state({});

	const parser = createParser({
		modules: [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		],
		fieldParsers: {},
		customValidators: {}
	});

	function getMainAccount(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].account;
		}
		return entidade.account || '';
	}

	function getMainValue(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].amount?.value;
		}
		return entidade.amount?.value;
	}

	function getMainCurrency(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].amount?.currency;
		}
		return entidade.amount?.currency;
	}

	function getNarration(entidade: any) {
		return entidade.narration || entidade.comment || '';
	}

	function getTags(entidade: any) {
		return entidade.tags || [];
	}

	function getPostingType(posting: any, entidade: any) {
		// Para transações, o tipo pode ser flag ou keyword
		if (entidade.flag) return entidade.flag;
		if (entidade.keyword) return entidade.keyword;
		return '';
	}

	function toggleCollapse(idx: number) {
		openCollapse = { ...openCollapse, [idx]: !openCollapse[idx] };
	}

	$effect(() => {
		console.log(files);
		if (!files) return;
		if (files.length != 1) return;
		files
			.item(0)
			?.text()
			.then((text: string) => {
				try {
					content = parser(text);
					erro = null;
				} catch (e: unknown) {
					erro = e instanceof Error ? e.message : String(e);
					content = [];
				}
			});
		console.log(content);
	});
</script>

<style>
.valor {
	text-align: right;
	font-variant-numeric: tabular-nums;
}
</style>

<h1>Beancount preview</h1>

<input type="file" bind:files />
{#if erro != null}
	<p><b>Erro: </b>: {erro}</p>
{/if}

{#if content.length > 0}
	<Table striped hover responsive>
		<thead>
			<tr>
				<th>Data</th>
				<th>Tipo</th>
				<th>Descrição</th>
				<th>Detalhes</th>
			</tr>
		</thead>
		<tbody>
			{#each content as entidade, i}
				<tr>
					<td>{entidade.date}</td>
					<td>{entidade.kind}</td>
					<td>
						{#if entidade.kind === 'balance'}
							<strong>{entidade.account}</strong>
							{#if entidade.amount}
								&nbsp;{entidade.amount.value} {entidade.amount.currency}
							{/if}
						{:else}
							{#if entidade.payee}
								<strong>{entidade.payee}</strong>
							{/if}
							{#if entidade.narration}
								{#if entidade.payee} {entidade.narration}{:else}{entidade.narration}{/if}
							{:else if entidade.comment}
								{entidade.comment}
							{/if}
						{/if}
						<div style="margin-top: 0.25em;">
							{#each getTags(entidade) as tag}
								<Badge color="secondary" class="me-1">{tag}</Badge>
							{/each}
						</div>
					</td>
					<td>
						<Button color="primary" size="sm" on:click={() => toggleCollapse(i)}>
							{openCollapse[i] ? 'Ocultar' : 'Ver detalhes'}
						</Button>
					</td>
				</tr>
				<tr>
					<td colspan="4" style="padding:0; border: none; background: transparent;">
						<Collapse isOpen={!!openCollapse[i]}>
							<div style="padding: 1rem; background: #f8f9fa; border-radius: 0 0 0.5rem 0.5rem; border: 1px solid #dee2e6; border-top: none;">
								{#if entidade.kind === 'transaction' && entidade.postings}
									<ListGroup class="mb-2">
										{#each entidade.postings as posting}
											<ListGroupItem>
												<div style="display: flex; justify-content: space-between; align-items: center;">
													<span style="font-weight: 500;">{posting.account}</span>
													<span style="font-variant-numeric: tabular-nums; text-align: right; min-width: 100px;">{posting.amount?.value ?? ''} {posting.amount?.currency ?? ''}</span>
												</div>
												{#if posting.meta}
													<br /><em>Meta:</em>
													<ListGroup class="mt-1">
														{#each Object.entries(posting.meta) as [k, v]}
															<ListGroupItem>{k}: {JSON.stringify(v)}</ListGroupItem>
														{/each}
													</ListGroup>
												{/if}
											</ListGroupItem>
										{/each}
									</ListGroup>
								{/if}
								{#if entidade.meta}
									<strong>Meta:</strong>
									<ListGroup class="mb-2">
										{#each Object.entries(entidade.meta) as [k, v]}
											<ListGroupItem>{k}: {JSON.stringify(v)}</ListGroupItem>
										{/each}
									</ListGroup>
								{/if}
							</div>
						</Collapse>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{:else}
	<p>Nenhuma entidade encontrada.</p>
{/if}
