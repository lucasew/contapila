// Wrapper para Web Worker do parser
export class ParserWorker {
  private worker: Worker | null = null;
  private messageId = 0;
  private pendingMessages = new Map<number, { resolve: Function; reject: Function }>();

  constructor() {
    if (typeof window !== 'undefined') {
      try {
        this.worker = new Worker(new URL('./worker.js', import.meta.url), { type: 'module' });
        this.worker.addEventListener('message', this.handleMessage.bind(this));
      } catch (error) {
        console.error('Erro ao criar Web Worker:', error);
        throw error;
      }
    }
  }

  private handleMessage(event: MessageEvent) {
    const { id, type, data } = event.data;
    
    // Trata mensagens de progresso (não precisam de pending)
    if (type === 'progress') {
      return;
    }
    
    const pending = this.pendingMessages.get(id);
    
    if (!pending) {
      return;
    }
    
    this.pendingMessages.delete(id);
    
    if (type === 'success') {
      pending.resolve(data);
    } else if (type === 'error') {
      pending.reject(new Error(data.message));
    }
  }

  private sendMessage(type: string, data: any): Promise<any> {
    return new Promise((resolve, reject) => {
      if (!this.worker) {
        reject(new Error('Worker não disponível'));
        return;
      }

      const id = ++this.messageId;
      this.pendingMessages.set(id, { resolve, reject });
      
      this.worker.postMessage({ id, type, data });
    });
  }

  async parseFile(text: string, filename: string): Promise<any[]> {
    return this.sendMessage('parse', { text, filename });
  }

  async parseMultipleFiles(files: Array<{ text: string; filename: string }>): Promise<Array<{ success: boolean; entries: any[]; error: string | null }>> {
    return this.sendMessage('parseMultiple', { files });
  }

  terminate() {
    if (this.worker) {
      this.worker.terminate();
      this.worker = null;
    }
  }
} 