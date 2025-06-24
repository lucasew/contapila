import { writable } from 'svelte/store';

interface ProcessingState {
  isProcessing: boolean;
}

function createProcessingStore() {
  const { subscribe, set } = writable<ProcessingState>({ isProcessing: false });

  return {
    subscribe,
    startProcessing: () => set({ isProcessing: true }),
    stopProcessing: () => set({ isProcessing: false }),
  };
}

export const processingStore = createProcessingStore(); 