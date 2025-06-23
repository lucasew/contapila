import { describe, test, expect, beforeEach } from 'vitest';
import {
  REGEX_PATTERNS,
  createParserConfig,
  createParser,
  parseAmount,
  convertBeancountToGeneralizedFormat,
  validateModuleDependencies,
  parseTagOrLink,
  parseTagsAndLinks
} from './parser.js';
import { createCoreBeancountModule } from './beancount.js';

// Utilitário de teste para criar um cursor
const createTestCursor = (text: string, position = 0) => ({
  text,
  position,
  line: 1,
  column: 1
});

// Os testes serão movidos do beancount.spec.ts para cá. 

describe('Regex Patterns', () => {
	test('DATE pattern matches valid dates', () => {
		expect('2024-01-01'.match(REGEX_PATTERNS.DATE)).toBeTruthy();
		expect('2024-12-31'.match(REGEX_PATTERNS.DATE)).toBeTruthy();
		expect('invalid-date'.match(REGEX_PATTERNS.DATE)).toBeFalsy();
		expect('24-01-01'.match(REGEX_PATTERNS.DATE)).toBeFalsy();
	});

	test('ACCOUNT pattern matches valid accounts', () => {
		expect('Assets:Cash'.match(REGEX_PATTERNS.ACCOUNT)).toBeTruthy();
		expect('Expenses:Food:Groceries'.match(REGEX_PATTERNS.ACCOUNT)).toBeTruthy();
		expect('Income:Salary_2024'.match(REGEX_PATTERNS.ACCOUNT)).toBeTruthy();
		expect('assets:cash'.match(REGEX_PATTERNS.ACCOUNT)).toBeFalsy(); // lowercase first letter
	});
});

describe('Core Parser Functions', () => {
	describe('Cursor utilities', () => {
		test('createCursor initializes corretamente', () => {
			const cursor = createTestCursor('test text');
			expect(cursor.text).toBe('test text');
			expect(cursor.position).toBe(0);
			expect(cursor.line).toBe(1);
			expect(cursor.column).toBe(1);
		});
	});

	describe('Basic parsers', () => {
		test('parseAmount extrai valores com moedas', () => {
			const config = createParserConfig([]);
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
	});
});

describe('YAML Parser', () => {
	test('parses simple key-value pairs', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);

		const entryWithMeta = `2024-01-01 open Assets:Cash USD\n  description: \"Main account\"\n  priority: high\n  active: true\n  balance: 1000.50`;

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

		const entryWithMeta = `2024-01-01 open Assets:Cash USD\n  string_value: test\n  integer_value: 42\n  float_value: 3.14\n  boolean_true: true\n  boolean_false: false\n  null_value: `;

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

describe('Funções utilitárias do parser', () => {
	test('convertBeancountToGeneralizedFormat transforma corretamente', () => {
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

	test('validateModuleDependencies detecta dependências faltantes', () => {
		const modules: any[] = [
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

	test('validateModuleDependencies passa com dependências válidas', () => {
		const modules: any[] = [
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
});

describe('Linha com ;; e string', () => {
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
		});
	});
});

describe('Diretivas open com ;;, string e metadados', () => {
	test('parses open with ;; and string', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = '2015-03-01 open Assets:XXX:YYY ;; "SOME_STRING"';
		const entries = parser(text);
		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:XXX:YYY',
		});
	});

	test('parses open with ;;, string and meta', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = `2015-03-01 open Assets:XXX:YYY:AAA ;; "SOME_STRING"\n  meta1: "TRUE"\n  meta2: "SOMETHING"\n  logic: "custom_logic"`;
		const entries = parser(text);
		expect(entries).toHaveLength(1);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:XXX:YYY:AAA',
		});
		expect(entries[0].meta).toMatchObject({
			meta1: 'TRUE',
			meta2: 'SOMETHING',
			logic: 'custom_logic'
		});
	});

	test('parses multiple open with ;;, string and meta', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = `2015-03-01 open Assets:XXX:YYY:AAA ;; "SOME_STRING"\n  meta1: "TRUE"\n  meta2: "SOMETHING"\n  logic: "custom_logic"\n\n2015-03-01 open Assets:XXX:YYY:BBB ;; "SOME_STRING"\n  meta1: "TRUE"\n  meta2: "SOMETHING"\n  logic: "custom_logic2"`;
		const entries = parser(text);
		expect(entries).toHaveLength(2);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:XXX:YYY:AAA',
		});
		expect(entries[0].meta).toMatchObject({
			meta1: 'TRUE',
			meta2: 'SOMETHING',
			logic: 'custom_logic'
		});
		expect(entries[1]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:XXX:YYY:BBB',
		});
		expect(entries[1].meta).toMatchObject({
			meta1: 'TRUE',
			meta2: 'SOMETHING',
			logic: 'custom_logic2'
		});
	});
});

describe('Teste anonimizado de open com ;;, string e metadados', () => {
	test('parses múltiplas diretivas open com ;;, string e metadados anonimizados', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = `2015-03-01 open Assets:AAA:BBB ;; "STRING1"
\n2015-03-01 open Assets:AAA:BBB:CCC ;; "STRING2"
  meta_a: "TRUE"
  meta_b: "VALOR1"
  logic: "logica1"
\n2015-03-01 open Assets:AAA:BBB:DDD ;; "STRING3"
  meta_a: "TRUE"
  meta_b: "VALOR2"
  logic: "logica2"`;
		const entries = parser(text);
		expect(entries).toHaveLength(3);
		expect(entries[0]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:AAA:BBB',
		});
		expect(entries[1]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:AAA:BBB:CCC',
		});
		expect(entries[1].meta).toMatchObject({
			meta_a: 'TRUE',
			meta_b: 'VALOR1',
			logic: 'logica1'
		});
		expect(entries[2]).toMatchObject({
			kind: 'open',
			date: '2015-03-01',
			account: 'Assets:AAA:BBB:DDD',
		});
		expect(entries[2].meta).toMatchObject({
			meta_a: 'TRUE',
			meta_b: 'VALOR2',
			logic: 'logica2'
		});
	});
});

describe('Parsing de tags (#) e links (^) nas diretivas', () => {
	test('parseTagOrLink reconhece tag', () => {
		const cursor = createTestCursor('#tag1');
		const result = parseTagOrLink(cursor);
		expect(result).not.toBeNull();
		if (result) expect(result.value).toEqual({ type: 'tag', value: 'tag1' });
	});
	test('parseTagOrLink reconhece link', () => {
		const cursor = createTestCursor('^link1');
		const result = parseTagOrLink(cursor);
		expect(result).not.toBeNull();
		if (result) expect(result.value).toEqual({ type: 'link', value: 'link1' });
	});
	test('parseTagsAndLinks reconhece tags e links misturados', () => {
		const cursor = createTestCursor('#tag1 ^link1 #tag2 ^link2');
		const result = parseTagsAndLinks(cursor);
		expect(result).not.toBeNull();
		if (result) expect(result.value).toEqual([
			{ type: 'tag', value: 'tag1' },
			{ type: 'link', value: 'link1' },
			{ type: 'tag', value: 'tag2' },
			{ type: 'link', value: 'link2' }
		]);
	});
	test('parseTagsAndLinks só links', () => {
		const cursor = createTestCursor('^linkA ^linkB');
		const result = parseTagsAndLinks(cursor);
		expect(result).not.toBeNull();
		if (result) expect(result.value).toEqual([
			{ type: 'link', value: 'linkA' },
			{ type: 'link', value: 'linkB' }
		]);
	});
	test('parseDirective popula tags e links separadamente', () => {
		const config = createParserConfig([createCoreBeancountModule()]);
		const parser = createParser(config);
		const text = '2024-01-01 open Assets:Cash USD #tag1 ^link1 #tag2';
		const entries = parser(text);
		expect(entries).toHaveLength(1);
		expect(entries[0].tags).toEqual(['tag1', 'tag2']);
		expect(entries[0].links).toEqual(['link1']);
	});
}); 