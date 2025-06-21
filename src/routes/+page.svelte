<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createParser,
		createTransactionModule
	} from '$lib/beancount.js';

	let files = $state();
	let content = $state([]);
	$effect(() => {
		console.log(files);
		if (!files) return;
		if (files.length != 1) return;
		files
			.item('utf-8')
			.text()
			.then((text) => (content = parser(text)));
	});
	const parser = createParser({
		modules: [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		],
		fieldParsers: {},
		customValidators: {}
	});
</script>

<h1>Beancount preview</h1>

<input type="file" bind:files />

<pre>{content}</pre>
