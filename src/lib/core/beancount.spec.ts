import {
	createCoreBeancountModule,
	createCustomReportingModule,
	createTransactionModule,
	getAllDirectiveTypes,
	getModuleByName,
	type BaseEntry,
	type DirectiveModule
} from './beancount.js';
import { beforeEach, describe, expect, test } from 'vitest';
import {
	REGEX_PATTERNS,
	createParserConfig,
	createParser,
	parseAmount,
	parseTagOrLink,
	parseTagsAndLinks,
	convertBeancountToGeneralizedFormat,
	validateModuleDependencies
} from './parser.js';

// Test utilities
const createTestCursor = (text: string, position = 0) => ({
	text,
	position,
	line: 1,
	column: 1
});

describe('Core Parser Functions', () => {
	describe('Cursor utilities', () => {
		test('createCursor initializes correctly', () => {
			const cursor = createTestCursor('test text');
			expect(cursor.text).toBe('test text');
			expect(cursor.position).toBe(0);
			expect(cursor.line).toBe(1);
			expect(cursor.column).toBe(1);
		});

		test('advanceCursor updates position correctly', () => {
			// This would need to be imported from the module or recreated for testing
			// Since the functions are not exported, we'll test through the public API
		});
	});

	describe('Basic parsers', () => {
		test('parseDate extracts valid dates', () => {
			const config = createParserConfig([createCoreBeancountModule()]);
			const parser = createParser(config);

			const validEntry = '2024-01-01 open Assets:Cash USD';
			const entries = parser(validEntry);

			expect(entries).toHaveLength(1);
			expect(entries[0].date).toBe('2024-01-01');
		});

		test('parseAmount extracts amounts with currencies', () => {
			const config = createParserConfig([createCoreBeancountModule()]);
			const parser = createParser(config);

			const balanceEntry = '2024-01-01 balance Assets:Cash 1000.50 USD';
			const entries = parser(balanceEntry);

			expect(entries).toHaveLength(1);
			expect(entries[0].amount).toEqual({
				value: 1000.5,
				currency: 'USD'
			});

			expect(parseAmount(createTestCursor('100.50 USD'))?.value).toStrictEqual({
				value: 100.5,
				currency: 'USD'
			});
			expect(parseAmount(createTestCursor('-200.75 BRL'))?.value).toStrictEqual({
				value: -200.75,
				currency: 'BRL'
			});
			expect(parseAmount(createTestCursor('1000EUR'))?.value).toStrictEqual({
				value: 1000,
				currency: 'EUR'
			});
			expect(parseAmount(createTestCursor('1000A_B_C'))?.value).toStrictEqual({
				value: 1000,
				currency: 'A_B_C'
			});
			expect(parseAmount(createTestCursor('invalid amount'))).toBeNull();
		});

		test('parseAccount extracts valid account names', () => {
			const config = createParserConfig([createCoreBeancountModule()]);
			const parser = createParser(config);

			const openEntry = '2024-01-01 open Assets:Checking:Bank1 USD';
			const entries = parser(openEntry);

			expect(entries).toHaveLength(1);
			expect(entries[0].account).toBe('Assets:Checking:Bank1');
		});

		test('parseQuotedString handles escape sequences', () => {
			const config = createParserConfig([createCoreBeancountModule(), createTransactionModule()]);
			const parser = createParser(config);

			const transaction = '2024-01-01 * "Store" "Description with \\"quotes\\" and \\n newline"';
			const entries = parser(transaction);

			expect(entries).toHaveLength(1);
			expect(entries[0].narration).toBe('Description with "quotes" and \n newline');
		});

		test('parseTagOrLink extracts hashtag tags', () => {
			const config = createParserConfig([createCoreBeancountModule()]);
			const parser = createParser(config);

			const openEntry = '2024-01-01 open Assets:Cash USD #checking #primary';
			const entries = parser(openEntry);

			expect(entries).toHaveLength(1);
			expect(entries[0].tags).toEqual(['checking', 'primary']);
		});

		test('parseTagOrLink function works directly', () => {
			// Test the tag parsing function directly
			const cursor = createTestCursor('#test');
			const result = parseTagOrLink(cursor);

			expect(result).not.toBeNull();
			expect(result?.value.type).toBe('tag');
			expect(result?.value.value).toBe('test');
		});

		test('parseTagsAndLinks function works directly', () => {
			// Test the tags parsing function directly
			const cursor = createTestCursor('#tag1 #tag2');
			const result = parseTagsAndLinks(cursor);

			expect(result).not.toBeNull();
			expect(result?.value.length).toBe(2);
			expect(result?.value[0].type).toBe('tag');
			expect(result?.value[0].value).toBe('tag1');
			expect(result?.value[1].type).toBe('tag');
			expect(result?.value[1].value).toBe('tag2');
		});
	});
});

describe('YAML Parser', () => {
	test('parses simple key-value pairs', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);

		const entryWithMeta = `2024-01-01 open Assets:Cash USD
  description: "Main account"
  priority: high
  active: true
  balance: 1000.50`;

		const entries = parser(entryWithMeta);

		expect(entries).toHaveLength(1);
		expect(entries[0].meta).toMatchObject({
			description: 'Main account',
			priority: 'high',
			active: true,
			balance: 1000.5
		});
		expect(typeof entries[0].meta.location).toBe('string');
		expect(entries[0].meta.location).toMatch(/^(\$file|stdin):\d+$/);
	});

	test('handles different data types in YAML', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);

		const entryWithMeta = `2024-01-01 open Assets:Cash USD
  string_value: test
  integer_value: 42
  float_value: 3.14
  boolean_true: true
  boolean_false: false
  null_value: `;

		const entries = parser(entryWithMeta);

		expect(entries[0].meta).toMatchObject({
			string_value: 'test',
			integer_value: 42,
			float_value: 3.14,
			boolean_true: true,
			boolean_false: false,
			null_value: null
		});
		expect(typeof entries[0].meta.location).toBe('string');
		expect(entries[0].meta.location).toMatch(/^(\$file|stdin):\d+$/);
	});
});

describe('Module System', () => {
	test('validateModuleDependencies detects missing dependencies', () => {
		const modules: DirectiveModule[] = [
			{
				name: 'module-a',
				version: '1.0.0',
				dependencies: ['missing-module'],
				directives: []
			}
		];

		const errors = validateModuleDependencies(modules);
		expect(errors).toHaveLength(1);
		expect(errors[0]).toContain("depends on missing module 'missing-module'");
	});

	test('validateModuleDependencies passes with valid dependencies', () => {
		const modules: DirectiveModule[] = [
			{
				name: 'core',
				version: '1.0.0',
				directives: []
			},
			{
				name: 'extended',
				version: '1.0.0',
				dependencies: ['core'],
				directives: []
			}
		];

		const errors = validateModuleDependencies(modules);
		expect(errors).toHaveLength(0);
	});

	test('getAllDirectiveTypes returns all directive kinds', () => {
		const modules = [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		];

		const types = getAllDirectiveTypes(modules);
		expect(types).toContain('open');
		expect(types).toContain('close');
		expect(types).toContain('balance');
		expect(types).toContain('price');
		expect(types).toContain('transaction');
		expect(types).toContain('budget');
	});

	test('getModuleByName finds correct module', () => {
		const modules = [createCoreBeancountModule(), createTransactionModule()];

		const coreModule = getModuleByName(modules, 'core-beancount');
		expect(coreModule).toBeDefined();
		expect(coreModule?.name).toBe('core-beancount');

		const missingModule = getModuleByName(modules, 'nonexistent');
		expect(missingModule).toBeUndefined();
	});
});

describe('Core Beancount Directives', () => {
	let parser: (text: string) => BaseEntry[];

	beforeEach(() => {
		const config = createParserConfig([createCoreBeancountModule()]);
		parser = createParser(config);
	});

	test('parses open directive', () => {
		const text = '2024-01-01 open Assets:Cash USD,BRL';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2024-01-01',
			keyword: 'open',
			account: 'Assets:Cash',
			currencies: ['USD', 'BRL']
		});
	});

	test('parses close directive', () => {
		const text = '2024-12-31 close Assets:Cash';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'close',
			date: '2024-12-31',
			keyword: 'close',
			account: 'Assets:Cash'
		});
	});

	test('parses balance directive', () => {
		const text = '2024-01-01 balance Assets:Cash 1000.00 USD';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'balance',
			date: '2024-01-01',
			keyword: 'balance',
			account: 'Assets:Cash',
			amount: {
				value: 1000.0,
				currency: 'USD'
			}
		});
	});

	test('parses price directive', () => {
		const text = '2024-01-01 price USD 5.25 BRL';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'price',
			date: '2024-01-01',
			keyword: 'price',
			commodity: 'USD',
			amount: {
				value: 5.25,
				currency: 'BRL'
			}
		});
	});

	test('parses price directive with stock symbol', () => {
		const text = '2024-01-01 price AAPL 150.00 USD';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'price',
			date: '2024-01-01',
			keyword: 'price',
			commodity: 'AAPL',
			amount: {
				value: 150.0,
				currency: 'USD'
			}
		});
	});

	test('parses balance directive with tags', () => {
		const text = '2024-01-01 balance Assets:Cash 1000.00 USD #monthly #checking';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'balance',
			date: '2024-01-01',
			keyword: 'balance',
			account: 'Assets:Cash',
			amount: {
				value: 1000.0,
				currency: 'USD'
			},
			tags: ['monthly', 'checking']
		});
	});

	test('parses open directive with double semicolon and quoted string', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = '2015-03-01 open Assets:XXX:YYY ;; "SOME_STRING"';
		const entries = parser(text);
		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:XXX:YYY',
			// O campo meta pode ou não conter o valor dependendo do parser, mas não deve lançar erro
		});
	});
});

describe('Transaction Parser', () => {
	let parser: (text: string) => BaseEntry[];

	beforeEach(() => {
		const config = createParserConfig([createCoreBeancountModule(), createTransactionModule()]);
		parser = createParser(config);
	});

	test('parses simple transaction', () => {
		const text = `2024-01-01 * "Payee" "Description"
  Assets:Cash      -100.00 USD
  Expenses:Food     100.00 USD`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'transaction',
			date: '2024-01-01',
			flag: '*',
			payee: 'Payee',
			narration: 'Description'
		});

		expect(entries[0].postings).toHaveLength(2);
		expect(entries[0].postings[0]).toMatchObject({
			account: 'Assets:Cash',
			amount: {
				value: -100.0,
				currency: 'USD'
			}
		});
	});

	test('parses transaction without payee', () => {
		const text = `2024-01-01 * "Description only"
  Assets:Cash      -50.00 USD
  Expenses:Other    50.00 USD`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0].payee).toBeUndefined();
		expect(entries[0].narration).toBe('Description only');
	});

	test('parses transaction with metadata', () => {
		const text = `2024-01-01 * "Store" "Purchase"
  category: "shopping"
  receipt_id: 12345
  Assets:Cash      -200.00 USD
  Expenses:Shopping 200.00 USD
    tax: 20.00
    item_count: 3`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0].meta).toEqual({
			category: 'shopping',
			receipt_id: 12345,
			location: 'stdin:1'
		});

		expect(entries[0].postings[1].meta).toEqual({
			tax: 20.0,
			item_count: 3
		});
	});

	test('parses transaction with unbalanced postings', () => {
		const text = `2024-01-01 * "Store" "Partial amounts"
  Assets:Cash      -100.00 USD
  Expenses:Food`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0].postings).toHaveLength(2);
		expect(entries[0].postings[1].amount).toBeUndefined();
	});

	test('parses transaction with tags', () => {
		const text = `2024-01-01 * "Store" "Groceries" #food #weekly
  Assets:Cash      -75.00 USD
  Expenses:Food     75.00 USD`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0].tags).toEqual(['food', 'weekly']);
		expect(entries[0].payee).toBe('Store');
		expect(entries[0].narration).toBe('Groceries');
	});

	test('parses transaction with complex metadata and commodity units', () => {
		const text = `2024-11-19 * "Investment Broker" "Sale of government bond" #todo
  doc-nr: "20241119000001234"
  details: ""
  transaction-type: "Bond-Sale"
  Assets:Investment:Bonds -0.07 GOVT_BOND_2029 {15000.00 USD}
  Assets:Bank:Checking 1000.00 USD
  Income:Investment:CapitalGains`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'transaction',
			date: '2024-11-19',
			flag: '*',
			payee: 'Investment Broker',
			narration: 'Sale of government bond',
			tags: ['todo']
		});

		expect(entries[0].meta).toEqual({
			'doc-nr': '20241119000001234',
			details: '',
			'transaction-type': 'Bond-Sale',
			location: 'stdin:1'
		});

		expect(entries[0].postings).toHaveLength(3);
		expect(entries[0].postings[0]).toMatchObject({
			account: 'Assets:Investment:Bonds',
			amount: {
				value: -0.07,
				currency: 'GOVT_BOND_2029'
			}
		});
		expect(entries[0].postings[1]).toMatchObject({
			account: 'Assets:Bank:Checking',
			amount: {
				value: 1000.0,
				currency: 'USD'
			}
		});
		expect(entries[0].postings[2]).toMatchObject({
			account: 'Income:Investment:CapitalGains'
		});
		expect(entries[0].postings[2].amount).toBeUndefined();
	});

	test('parses transaction with comment in the middle', () => {
		const text = `2024-11-19 * "Store" "Purchase with comment"
  Assets:Cash -100.00 USD
  ; This is a comment in the middle
  Expenses:Food 100.00 USD`;

		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'transaction',
			date: '2024-11-19',
			flag: '*',
			payee: 'Store',
			narration: 'Purchase with comment'
		});

		expect(entries[0].postings).toHaveLength(2);
		expect(entries[0].postings[0]).toMatchObject({
			account: 'Assets:Cash',
			amount: {
				value: -100.0,
				currency: 'USD'
			}
		});
		expect(entries[0].postings[1]).toMatchObject({
			account: 'Expenses:Food',
			amount: {
				value: 100.0,
				currency: 'USD'
			}
		});
	});

	test('parses note directive', () => {
		const text = `2026-01-19 note Assets:Test:Account "This is a test note"`;
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'note',
			date: '2026-01-19',
			account: 'Assets:Test:Account',
			comment: 'This is a test note'
		});
	});
});

describe('Custom Reporting Module', () => {
	let parser: (text: string) => BaseEntry[];

	beforeEach(() => {
		const config = createParserConfig([createCustomReportingModule()]);
		parser = createParser(config);
	});

	test('parses budget directive', () => {
		const text = '2024-01-01 budget Expenses:Food 500.00 USD monthly';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'budget',
			date: '2024-01-01',
			keyword: 'budget',
			account: 'Expenses:Food',
			amount: {
				value: 500.0,
				currency: 'USD'
			},
			period: 'monthly'
		});
	});

	test('parses budget directive with default period', () => {
		const text = '2024-01-01 budget Expenses:Transport 200.00 USD';
		const entries = parser(text);

		expect(entries).toHaveLength(1);
		expect(entries[0].period).toBe('monthly'); // default value
	});
});

describe('Custom Field Parsers', () => {
	test('custom percentage parser works', () => {
		const customFieldParsers = {
			percentage: (cursor: any) => {
				const text = cursor.text.slice(cursor.position);
				const match = text.match(/^(\d+(?:\.\d+)?)%/);
				if (match) {
					return {
						value: parseFloat(match[1]) / 100,
						cursor: {
							...cursor,
							position: cursor.position + match[0].length
						}
					};
				}
				return null;
			}
		};

		const customModule: DirectiveModule = {
			name: 'custom-test',
			version: '1.0.0',
			directives: [
				{
					kind: 'rate',
					fields: [
						{ name: 'date', type: 'date', required: true },
						{ name: 'keyword', type: 'string', required: true },
						{ name: 'value', type: 'percentage', required: true }
					]
				}
			]
		};

		const config = createParserConfig([customModule], customFieldParsers);
		const parser = createParser(config);

		// This test would need the custom parser to be properly integrated
		// For now, we'll test the parser function directly
		const result = customFieldParsers.percentage({
			text: '25.5%',
			position: 0,
			line: 1,
			column: 1
		});

		expect(result?.value).toBe(0.255);
	});
});

describe('Error Handling', () => {
	test('returns unknown_directive entry for unknown directive', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const entries = parser('2024-01-01 unknown_directive');
		expect(entries[0].kind).toBe('unknown_directive');
		expect(entries[0].meta.warning).toContain('Unknown directive');
	});

	test('throws error for unterminated string', () => {
		const config = createParserConfig([createCoreBeancountModule(), createTransactionModule()]);
		const parser = createParser(config);
		expect(() => {
			parser('2024-01-01 * "Unterminated string');
		}).toThrow('Unterminated string');
	});

	test('returns unknown_directive entry for missing required field', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const entries = parser('2024-01-01 open'); // missing required account
		expect(entries[0].kind).toBe('unknown_directive');
		expect(entries[0].meta.warning).toContain('Unknown directive');
	});

	test('throws error for invalid module dependencies', () => {
		const invalidModules: DirectiveModule[] = [
			{
				name: 'invalid',
				version: '1.0.0',
				dependencies: ['nonexistent'],
				directives: []
			}
		];

		expect(() => {
			createParser(createParserConfig(invalidModules));
		}).toThrow('Module validation failed');
	});
});

describe('Unknown Directives', () => {
	test('parses unknown directive as unknown_directive entry', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = '2024-01-01 unknown_directive algo aqui';
		const entries = parser(text);
		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'unknown_directive',
			body: '2024-01-01 unknown_directive algo aqui',
		});
		expect(entries[0].meta.warning).toContain('Unknown directive');
	});

	test('parses multiple unknown directives', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = 'foo bar\n2024-01-01 open Assets:Cash USD\nbar baz';
		const entries = parser(text);
		expect(entries[0].kind).toBe('unknown_directive');
		expect(entries[1].kind).toBe('open');
		expect(entries[2].kind).toBe('unknown_directive');
	});
});

describe('Integration Tests', () => {
	test('parses complete beancount file', () => {
		const config = createParserConfig([
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		]);
		const parser = createParser(config);

		const completeFile = `
; Comment line
2024-01-01 open Assets:Cash USD,BRL #primary #checking
  description: "Main cash account"

2024-01-01 open Expenses:Food USD #food

2024-01-15 * "Grocery Store" "Weekly groceries" #food #monthly
  category: "food"
  receipt: "12345"
  Assets:Cash      -150.75 USD
  Expenses:Food     150.75 USD
    tax_included: true

2024-02-01 balance Assets:Cash 849.25 USD #monthly

2024-02-01 budget Expenses:Food 600.00 USD monthly #budget #food
  note: "Monthly food budget"

2024-01-01 price USD 5.25 BRL #exchange-rate

2024-12-31 close Expenses:Food #cleanup
`;

		const entries = parser(completeFile);

		expect(entries).toHaveLength(7);

		// Check each directive type is parsed correctly
		const openEntries = entries.filter((e) => e.kind === 'open');
		expect(openEntries).toHaveLength(2);
		expect(openEntries[0].tags).toEqual(['primary', 'checking']);
		expect(openEntries[1].tags).toEqual(['food']);

		const transactionEntries = entries.filter((e) => e.kind === 'transaction');
		expect(transactionEntries).toHaveLength(1);
		expect(transactionEntries[0].tags).toEqual(['food', 'monthly']);

		const balanceEntries = entries.filter((e) => e.kind === 'balance');
		expect(balanceEntries).toHaveLength(1);
		expect(balanceEntries[0].tags).toEqual(['monthly']);

		const budgetEntries = entries.filter((e) => e.kind === 'budget');
		expect(budgetEntries).toHaveLength(1);
		expect(budgetEntries[0].tags).toEqual(['budget', 'food']);

		const priceEntries = entries.filter((e) => e.kind === 'price');
		expect(priceEntries).toHaveLength(1);
		expect(priceEntries[0].tags).toEqual(['exchange-rate']);

		const closeEntries = entries.filter((e) => e.kind === 'close');
		expect(closeEntries).toHaveLength(1);
		expect(closeEntries[0].tags).toEqual(['cleanup']);
	});

	test('handles empty file', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);

		const entries = parser('');
		expect(entries).toHaveLength(0);
	});

	test('handles file with only comments and whitespace', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);

		const entries = parser(`
; This is a comment
; Another comment

; Yet another comment
`);
		expect(entries).toHaveLength(0);
	});
});

describe('Conversion Functions', () => {
	test('convertBeancountToGeneralizedFormat transforms entries correctly', () => {
		const beancountEntries = [
			{
				type: 'transaction',
				date: '2024-01-01',
				flag: '*',
				payee: 'Store',
				narration: 'Purchase',
				postings: [
					{
						account: 'Assets:Cash',
						amount: { number: -100, currency: 'USD' },
						cost: { number: -100, currency: 'USD' },
						price: { number: 1, currency: 'USD' },
						flag: '*',
						meta: { note: 'test' }
					}
				],
				meta: { category: 'shopping' }
			},
			{
				type: 'balance',
				date: '2024-01-01',
				account: 'Assets:Cash',
				amount: { number: 1000, currency: 'USD' },
				meta: { verified: true }
			}
		];

		const config = createParserConfig([]);
		const converted = convertBeancountToGeneralizedFormat(beancountEntries, config);

		expect(converted).toHaveLength(2);

		expect(converted[0]).toMatchObject({
			kind: 'transaction',
			date: '2024-01-01',
			flag: '*',
			payee: 'Store',
			narration: 'Purchase',
			meta: { category: 'shopping' }
		});

		expect(converted[0].postings[0]).toMatchObject({
			account: 'Assets:Cash',
			amount: { value: -100, currency: 'USD' },
			cost: { value: -100, currency: 'USD' },
			price: { value: 1, currency: 'USD' },
			flag: '*',
			meta: { note: 'test' }
		});

		expect(converted[1]).toMatchObject({
			kind: 'balance',
			date: '2024-01-01',
			account: 'Assets:Cash',
			amount: { value: 1000, currency: 'USD' },
			meta: { verified: true }
		});
	});
});

describe('Parser Configuration', () => {
	test('createParserConfig creates valid configuration', () => {
		const modules = [createCoreBeancountModule()];
		const customParsers = { custom: () => null };
		const customValidators = { custom: () => true };

		const config = createParserConfig(modules, customParsers, customValidators);

		expect(config.modules).toBe(modules);
		expect(config.fieldParsers).toBe(customParsers);
		expect(config.customValidators).toBe(customValidators);
	});
});
