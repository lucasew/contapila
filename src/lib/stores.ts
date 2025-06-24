import { writable } from 'svelte/store';

interface ProcessingState {
  activeTasks: number;
}

function createProcessingStore() {
  const { subscribe, update } = writable<ProcessingState>({ activeTasks: 0 });

  async function runTask<T>(fn: () => Promise<T>): Promise<T> {
    update(state => ({ activeTasks: state.activeTasks + 1 }));
    try {
      return await fn();
    } finally {
      update(state => ({ activeTasks: Math.max(0, state.activeTasks - 1) }));
    }
  }

  return {
    subscribe,
    runTask,
  };
}

export const processingStore = createProcessingStore(); 