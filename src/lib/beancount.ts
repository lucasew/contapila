// Reexport your entry components here
// Constantes de regex no topo

import {
	REGEX_PATTERNS,
	advanceCursor,
	createParser,
	createParserConfig,
	isAtEnd,
	parseAccount,
	parseAmount,
	parseDate,
	parseNewline,
	parseQuotedString,
	parseRegex,
	parseString,
	parseTagOrLink,
	parseTagsAndLinks,
	parseYAML,
	peekChar,
	skipWhitespace,
	type AmountValue,
	type BaseEntry,
	type DirectiveDefinition,
	type DirectiveModule,
	type FieldDefinition
} from './parser.js';

// Módulos funcionais
const createCoreBeancountModule = (): DirectiveModule => ({
	name: 'core-beancount',
	version: '1.0.0',
	directives: [
		{
			kind: 'open',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'open')
				},
				{ name: 'account', type: 'account', required: true },
				{
					name: 'currencies',
					type: 'array',
					required: false,
					parser: (cursor) => {
						const result = parseRegex(cursor, REGEX_PATTERNS.CURRENCY);
						if (!result) return null;
						let current = result.cursor;
						const currencies = [result.value[1]];

						// Parse additional comma-separated currencies
						while (peekChar(current) === ',') {
							current = advanceCursor(current, 1); // Skip comma
							current = skipWhitespace(current);
							const nextResult = parseRegex(current, REGEX_PATTERNS.CURRENCY);
							if (!nextResult) break;
							currencies.push(nextResult.value[1]);
							current = nextResult.cursor;
						}

						return { value: currencies, cursor: current };
					}
				}
			]
		},
		{
			kind: 'close',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'close')
				},
				{ name: 'account', type: 'account', required: true }
			]
		},
		{
			kind: 'balance',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'balance')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'amount', type: 'amount', required: true }
			]
		},
		{
			kind: 'price',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'price')
				},
				{ name: 'commodity', type: 'string', required: true },
				{ name: 'amount', type: 'amount', required: true }
			]
		},
		{
			kind: 'note',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'note')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'comment', type: 'string', required: true, parser: parseQuotedString }
			]
		}
	]
});

const createTransactionModule = (): DirectiveModule => ({
	name: 'transactions',
	version: '1.0.0',
	dependencies: ['core-beancount'],
	directives: [
		{
			kind: 'transaction',
			customParser: (cursor, fields) => {
				let current = cursor;

				const dateResult = parseDate(current);
				if (!dateResult) return null;
				current = skipWhitespace(dateResult.cursor);

				const flagResult = parseRegex(current, REGEX_PATTERNS.FLAG);
				if (!flagResult) return null;
				current = skipWhitespace(flagResult.cursor);

				let payee: string | undefined;
				let narration: string;

				const firstQuotedResult = parseQuotedString(current);
				if (!firstQuotedResult) return null;
				current = skipWhitespace(firstQuotedResult.cursor);

				const secondQuotedResult = parseQuotedString(current);
				if (secondQuotedResult) {
					// Two quoted strings: first is payee, second is narration
					payee = firstQuotedResult.value;
					narration = secondQuotedResult.value;
					current = skipWhitespace(secondQuotedResult.cursor);
				} else {
					// One quoted string: it's the narration, no payee
					narration = firstQuotedResult.value;
				}

				// Parse tags at the end of the line
				let tags: string[] | undefined;
				let links: string[] | undefined;
				const tagsLinksResult = parseTagsAndLinks(current);
				if (tagsLinksResult) {
					tags = tagsLinksResult.value.filter(i => i.type === 'tag').map(i => i.value);
					links = tagsLinksResult.value.filter(i => i.type === 'link').map(i => i.value);
					current = tagsLinksResult.cursor;
				}

				// Após parsear tags e links, se encontrar ';;', ignora o resto da linha E FINALIZA a transação
				current = skipWhitespace(current);
				const restOfLine = current.text.slice(current.position).split('\n')[0];
				const semicolonIndex = restOfLine.indexOf(';;');
				let foundComment = false;
				if (semicolonIndex !== -1) {
					current = advanceCursor(current, semicolonIndex);
					while (!isAtEnd(current) && peekChar(current) !== '\n') {
						current = advanceCursor(current, 1);
					}
					foundComment = true;
				}

				const newlineResult = parseNewline(current);
				if (newlineResult) {
					current = newlineResult.cursor;
				}

				// Se encontrou ;;, finalize a transação aqui, sem postings nem metadados
				if (foundComment) {
					// Ignora linhas de postings órfãos imediatamente após a transação
					while (!isAtEnd(current)) {
						const firstChar = peekChar(current);
						const secondChar = peekChar(current, 1);
						const thirdChar = peekChar(current, 2);
						const fourthChar = peekChar(current, 3);
						if ((firstChar === ' ' && secondChar === ' ') ||
							(firstChar === ' ' && secondChar === ' ' && thirdChar === ' ' && fourthChar === ' ')) {
							// Avança até o fim da linha
							while (!isAtEnd(current) && peekChar(current) !== '\n') {
								current = advanceCursor(current, 1);
							}
							if (peekChar(current) === '\n') current = advanceCursor(current, 1);
						} else {
							break;
						}
					}
					return {
						value: {
							kind: 'transaction',
							date: dateResult.value,
							flag: flagResult.value[1],
							payee,
							narration,
							postings: [],
							meta: undefined,
							tags,
							links
						},
						cursor: current
					};
				}

				// Parse transaction content (metadata and postings)
				let meta: any = {};
				const postings: any[] = [];

				while (!isAtEnd(current)) {
					const indentResult = parseRegex(current, REGEX_PATTERNS.INDENT_TWO);
					if (!indentResult) break;
					current = indentResult.cursor;

					// Check if this line is a comment
					const commentResult = parseRegex(current, REGEX_PATTERNS.COMMENT);
					if (commentResult) {
						// Skip comment line
						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) {
							current = restOfLine.cursor;
						}
						const commentNewline = parseNewline(current);
						if (commentNewline) {
							current = commentNewline.cursor;
						}
						continue;
					}

					// Check if this line is metadata (YAML key-value)
					const yamlKeyCheck = parseRegex(current, REGEX_PATTERNS.YAML_KEY);
					if (yamlKeyCheck) {
						// Parse as metadata
						const key = yamlKeyCheck.value[1];
						current = yamlKeyCheck.cursor;

						let value: any;
						const quotedStr = parseQuotedString(current);
						if (quotedStr) {
							value = quotedStr.value;
							current = quotedStr.cursor;
						} else {
							const valueMatch = parseRegex(current, REGEX_PATTERNS.YAML_VALUE);
							if (valueMatch) {
								const rawValue = valueMatch.value[1].trim();
								if (REGEX_PATTERNS.INTEGER.test(rawValue)) value = parseInt(rawValue);
								else if (REGEX_PATTERNS.FLOAT.test(rawValue)) value = parseFloat(rawValue);
								else if (rawValue === 'true' || rawValue === 'false') value = rawValue === 'true';
								else value = rawValue || null;
								current = valueMatch.cursor;
							}
						}

						meta[key] = value;

						const metaNewline = parseNewline(current);
						if (metaNewline) {
							current = metaNewline.cursor;
						}
					} else {
						// Parse as posting
						const accountResult = parseAccount(current);
						if (!accountResult) break;
						current = skipWhitespace(accountResult.cursor);

						let amount: AmountValue | undefined;
						const amountResult = parseAmount(current);
						if (amountResult) {
							amount = amountResult.value;
							current = amountResult.cursor;
						}

						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) {
							current = restOfLine.cursor;
						}

						const postingNewline = parseNewline(current);
						if (postingNewline) {
							current = postingNewline.cursor;
						}

						let postingMeta: any;
						const postingMetaResult = parseYAML(current, 4);
						if (postingMetaResult) {
							postingMeta = postingMetaResult.value;
							current = postingMetaResult.cursor;
						}

						postings.push({
							account: accountResult.value,
							amount,
							meta: postingMeta || {}
						});
					}
				}

				return {
					value: {
						kind: 'transaction',
						date: dateResult.value,
						flag: flagResult.value[1],
						payee,
						narration,
						postings,
						meta: Object.keys(meta).length > 0 ? meta : {},
						tags,
						links
					},
					cursor: current
				};
			},
			fields: []
		}
	]
});

const createCustomReportingModule = (): DirectiveModule => ({
	name: 'custom-reporting',
	version: '1.0.0',
	directives: [
		{
			kind: 'budget',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'budget')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'amount', type: 'amount', required: true },
				{ name: 'period', type: 'string', required: false, defaultValue: 'monthly' }
			]
		}
	]
});

// Utilitários funcionais
const getAllDirectiveTypes = (modules: DirectiveModule[]): string[] =>
	modules.flatMap((module) => module.directives.map((directive) => directive.kind));

const getModuleByName = (modules: DirectiveModule[], name: string): DirectiveModule | undefined =>
	modules.find((module) => module.name === name);

// Utility: sum amounts by currency
function sumAmountsByCurrency(postings: { amount?: AmountValue }[]): Record<string, number> {
	const balance: Record<string, number> = {};
	for (const p of postings) {
		if (p.amount) {
			const { value, currency } = p.amount;
			balance[currency] = (balance[currency] || 0) + Number(value);
		}
	}
	return balance;
}

// Main validator/filler for transactions
function validateAndFillTransactions(entries: BaseEntry[]): { entries: BaseEntry[]; errors: { source: any; message: string; entry: BaseEntry }[] } {
	const errors: { source: any; message: string; entry: BaseEntry }[] = [];
	const newEntries = entries.map(entry => {
		if (entry.kind !== 'transaction') return entry;

		const postings = (entry.postings || []) as { amount?: AmountValue }[];
		const postingsWithoutAmount = postings.filter((p: { amount?: AmountValue }) => !p.amount);
		if (postingsWithoutAmount.length === 0) return entry; // Nothing to do

		if (postingsWithoutAmount.length === 1) {
			const balance = sumAmountsByCurrency(postings);
			const currencies = Object.keys(balance);
			if (currencies.length === 1) {
				const currency = currencies[0];
				const missingAmount = -balance[currency];
				const newPostings = postings.map((p: { amount?: AmountValue }) =>
					p.amount
						? p
						: { ...p, amount: { value: missingAmount, currency } }
				);
				return { ...entry, postings: newPostings };
			}
			// More than one currency: cannot infer
			errors.push({ source: entry.meta, message: 'Cannot infer missing amount: multiple currencies present.', entry });
			return entry;
		} else {
			// More than one posting without amount: error
			errors.push({ source: entry.meta, message: 'More than one posting without amount.', entry });
			return entry;
		}
	});
	return { entries: newEntries, errors };
}

// Exemplo de uso funcional
const exampleUsage = () => {
	const config = createParserConfig(
		[createCoreBeancountModule(), createTransactionModule(), createCustomReportingModule()],
		{
			// Custom field parser para percentual
			percentage: (cursor) => {
				const result = parseRegex(cursor, /^(\d+(?:\.\d+)?)%/);
				return result ? { value: parseFloat(result.value[1]) / 100, cursor: result.cursor } : null;
			}
		}
	);

	const parser = createParser(config);

	const testText = `
2024-01-01 open Assets:Cash USD,BRL #primary #checking #main
 description: "Conta principal"

2024-01-15 * "Mercado" "Compras do mês" #food #monthly #groceries
 category: "groceries"
 Assets:Cash      -200.00 BRL
 Expenses:Food     200.00 BRL

2024-02-01 budget Expenses:Food 500.00 BRL monthly #budget #food #monthly
 note: "Orçamento mensal para alimentação"

2024-01-01 price USD 5.25 BRL #exchange-rate #currency
2024-01-15 price AAPL 150.00 USD #stocks #tech #apple
`;

	const entries = parser(testText);
	const directiveTypes = getAllDirectiveTypes(config.modules);

	return { entries, directiveTypes };
};

export {
	createCoreBeancountModule,
	createCustomReportingModule,
	createTransactionModule,
	exampleUsage,
	getAllDirectiveTypes,
	getModuleByName,
	type AmountValue,
	type BaseEntry,
	type DirectiveDefinition,
	type DirectiveModule,
	type FieldDefinition,
	validateAndFillTransactions
};
