// Web Worker para processar parser em background
import { createParser } from '$lib/core/parser.js';
import { 
  createCoreBeancountModule, 
  createTransactionModule, 
  createCustomReportingModule 
} from '$lib/core/beancount.js';

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
  const { id, type, data } = event.data;
  
  try {
    switch (type) {
      case 'parse':
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
        self.postMessage({ id, type: 'success', data: entries });
        break;
        
      case 'parseMultiple':
        const { files } = data;
        const results = [];
        
        for (let i = 0; i < files.length; i++) {
          const { text, filename } = files[i];
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
        
        self.postMessage({ id, type: 'success', data: results });
        break;
        
      default:
        throw new Error(`Tipo de mensagem desconhecido: ${type}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    self.postMessage({ 
      id, 
      type: 'error', 
      data: { message: errorMessage } 
    });
  }
}); 