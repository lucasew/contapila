// Web Worker para processar parser em background
import { createParser } from './parser.js';
import { 
  createCoreBeancountModule, 
  createTransactionModule, 
  createCustomReportingModule 
} from './beancount.js';

console.log('Web Worker iniciado');

// Cria o parser no worker
const parser = createParser({
  modules: [
    createCoreBeancountModule(),
    createTransactionModule(),
    createCustomReportingModule()
  ],
  fieldParsers: {},
  customValidators: {}
});

// Listener para mensagens do main thread
self.addEventListener('message', async (event) => {
  console.log('Worker recebeu mensagem:', event.data);
  const { id, type, data } = event.data;
  
  try {
    switch (type) {
      case 'parse':
        console.log('Processando arquivo único:', data.filename);
        const { text, filename } = data;
        // Cria parser com filename específico
        const parserWithFilename = createParser({
          modules: [
            createCoreBeancountModule(),
            createTransactionModule(),
            createCustomReportingModule()
          ],
          fieldParsers: {},
          customValidators: {}
        }, filename);
        const entries = parserWithFilename(text);
        console.log('Arquivo processado com sucesso, enviando resposta');
        self.postMessage({ id, type: 'success', data: entries });
        break;
        
      case 'parseMultiple':
        console.log('Processando múltiplos arquivos:', data.files.length);
        const { files } = data;
        const results = [];
        
        for (let i = 0; i < files.length; i++) {
          const { text, filename } = files[i];
          console.log(`Processando arquivo ${i + 1}/${files.length}: ${filename}`);
          try {
            const parserWithFilename = createParser({
              modules: [
                createCoreBeancountModule(),
                createTransactionModule(),
                createCustomReportingModule()
              ],
              fieldParsers: {},
              customValidators: {}
            }, filename);
            const entries = parserWithFilename(text);
            results.push({ success: true, entries, error: null });
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error('Erro ao processar arquivo:', filename, errorMessage);
            results.push({ 
              success: false, 
              entries: [], 
              error: `Erro no arquivo ${filename}: ${errorMessage}` 
            });
          }
          
          // Reporta progresso
          self.postMessage({ 
            id, 
            type: 'progress', 
            data: { current: i + 1, total: files.length } 
          });
        }
        
        console.log('Todos os arquivos processados, enviando resposta final');
        self.postMessage({ id, type: 'success', data: results });
        break;
        
      default:
        throw new Error(`Tipo de mensagem desconhecido: ${type}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error('Erro no worker:', errorMessage);
    self.postMessage({ 
      id, 
      type: 'error', 
      data: { message: errorMessage } 
    });
  }
}); 