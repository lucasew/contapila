<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/beancount.js';
	import { createParser } from '$lib/parser.js';
	import { Table, Badge, Button, Collapse, ListGroup, ListGroupItem, Row, Col, Accordion, AccordionItem, Icon } from '@sveltestrap/sveltestrap';
	import PostingItem from '$lib/PostingItem.svelte';

	let files: FileList | undefined = $state();
	let content: any[] = $state([]);
	let erro: string | null = $state(null);
	let openCollapse: Record<number, boolean> = $state({});
	let openPostings: boolean[][] = [];

	const parser = createParser({
		modules: [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		],
		fieldParsers: {},
		customValidators: {}
	});

	let linhasTabela = $derived(content.map(entidadeParaLinhaTabela));

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

	function togglePosting(i: number, j: number) {
		// Garante nova referência para reatividade
		openPostings = openPostings.slice();
		openPostings[i] = (openPostings[i] || []).slice();
		openPostings[i][j] = !openPostings[i]?.[j];
	}

	function entidadeParaLinhaTabela(entidade: any) {
		let titulo = '';
		let subtitulo = '';
		if (entidade.kind === 'balance') {
			titulo = entidade.account;
			subtitulo = entidade.amount ? `${entidade.amount.value} ${entidade.amount.currency}` : '';
		} else if (entidade.kind === 'open' || entidade.kind === 'close') {
			titulo = entidade.account;
			subtitulo = '';
		} else {
			if (entidade.payee) titulo = entidade.payee;
			if (entidade.narration) subtitulo = entidade.narration;
			else if (entidade.comment) subtitulo = entidade.comment;
		}
		return {
			data: entidade.date,
			tipo: entidade.kind,
			titulo,
			subtitulo,
			detalhes: entidade
		};
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
		console.log($inspect(content));
	});
</script>




<input type="file" bind:files />
{#if erro != null}
	<p><b>Erro: </b>: {erro}</p>
{/if}

{#if linhasTabela.length > 0}
	<Accordion>
		{#each linhasTabela as linha, i}
			<AccordionItem>
				<Row slot="header" class="align-items-center w-100">
					<Col class="col-auto text-nowrap" >{linha.data}</Col>
					<Col class="col-auto text-nowrap" >{linha.tipo}</Col>
					<Col>
						{#if linha.titulo}
							<strong>{linha.titulo}</strong>
						{/if}
						{#if linha.subtitulo}
							{#if linha.titulo} {linha.subtitulo}{:else}{linha.subtitulo}{/if}
						{/if}
						
							{#each getTags(linha.detalhes) as tag}
								<Badge color="secondary" class="me-1">{tag}</Badge>
							{/each}
						
					</Col>
				</Row>
			
					{#if linha.detalhes.kind === 'transaction' && linha.detalhes.postings}
						{#each linha.detalhes.postings as posting, j}
							<PostingItem posting={posting} />
						{/each}
					{/if}
					{#if linha.detalhes.narration || linha.detalhes.comment || Object.keys(linha.detalhes).length > 0}
						<details class="mt-2">
							<summary><strong>Atributos</strong></summary>
							{#if linha.detalhes.meta}
								<ul class="ms-3 mb-2">
									{#each Object.entries(linha.detalhes.meta) as [k, v]}
										<li><strong>{k}:</strong> <span class="ms-3">{JSON.stringify(v)}</span></li>
									{/each}
								</ul>
							{:else}
								<span class="ms-3 text-muted">Sem atributos extras</span>
							{/if}
						</details>
					{/if}

			</AccordionItem>
		{/each}
	</Accordion>
{:else}
	<p>Nenhuma entidade encontrada.</p>
{/if}
