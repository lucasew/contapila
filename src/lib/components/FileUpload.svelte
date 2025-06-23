<script lang="ts">
import { createEventDispatcher } from 'svelte';
import { Form, FormGroup, Label, Input, Button, Icon, ListGroup, ListGroupItem, Row, Col, Badge } from '@sveltestrap/sveltestrap';

let files: File[] = [];
const dispatch = createEventDispatcher();
let isDragging = false;

function handleChange(e: Event) {
  const fileList = (e.target as HTMLInputElement).files;
  if (fileList) {
    const newFiles = Array.from(fileList);
    const names = new Set(files.map(f => f.name));
    files = files.concat(newFiles.filter(f => !names.has(f.name)));
    dispatch('change', { files: toFileList(files) });
  }
}

function clearFiles() {
  files = [];
  dispatch('change', { files: toFileList(files) });
  const input = document.getElementById('file-upload-input') as HTMLInputElement;
  if (input) input.value = '';
}

function handleDrop(e: DragEvent) {
  e.preventDefault();
  isDragging = false;
  if (e.dataTransfer?.files) {
    const newFiles = Array.from(e.dataTransfer.files);
    const names = new Set(files.map(f => f.name));
    files = files.concat(newFiles.filter(f => !names.has(f.name)));
    dispatch('change', { files: toFileList(files) });
    const input = document.getElementById('file-upload-input') as HTMLInputElement;
    if (input) input.value = '';
  }
}

function handleDragOver(e: DragEvent) {
  e.preventDefault();
  isDragging = true;
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault();
  isDragging = false;
}

function removeFile(idx: number) {
  files = files.slice(0, idx).concat(files.slice(idx + 1));
  dispatch('change', { files: toFileList(files) });
  const input = document.getElementById('file-upload-input') as HTMLInputElement;
  if (input) input.value = '';
}

function toFileList(arr: File[]): FileList {
  const dt = new DataTransfer();
  arr.forEach(f => dt.items.add(f));
  return dt.files;
}
</script>

<Form class="mb-3">
  <FormGroup>
    <Label for="file-upload-input" class="fw-bold mb-2">
      <Icon name="upload" class="me-2" />Upload de arquivos Beancount
    </Label>
    <Input
      id="file-upload-input"
      type="file"
      multiple
      on:change={handleChange}
      style="display: none;"
    />
    <ListGroup>
      <ListGroupItem
        class="w-100 p-4 mb-2 rounded border border-2 border-dashed d-flex flex-column align-items-center justify-content-center bg-white"
        style="min-height: 160px; cursor: pointer; transition: background 0.2s;"
        color={isDragging ? 'primary' : 'secondary'}
        on:click={() => document.getElementById('file-upload-input')?.click()}
        on:dragover={handleDragOver}
        on:dragleave={handleDragLeave}
        on:drop={handleDrop}
        active={isDragging}
      >
        <Icon name="upload" size="2x" class="mb-2 text-primary" />
        <div class="fw-bold mb-1">Arraste e solte arquivos aqui ou clique para selecionar</div>
        <div class="text-muted small">(Múltiplos arquivos suportados)</div>
        {#if files.length === 0}
          <div class="text-muted mt-2">Nenhum arquivo selecionado ainda.</div>
        {/if}
        {#if files.length > 0}
          <div class="w-100 mt-3">
            {#each files as file, idx}
              <Row class="align-items-center mb-2 g-2">
                <Col>
                  <span style="cursor: pointer;" on:click|stopPropagation={() => removeFile(idx)}>
                    <Badge color="primary" pill class="fs-6">
                      {file.name} <span class="text-muted small ms-1">({Math.round(file.size/1024)} KB)</span>
                    </Badge>
                  </span>
                </Col>
              </Row>
            {/each}
          </div>
          <div class="w-100 d-flex justify-content-center mt-2">
            <Button outline color="danger" on:click={clearFiles}>
              <Icon name="x" class="me-1" /> Limpar tudo
            </Button>
          </div>
        {/if}
      </ListGroupItem>
    </ListGroup>
  </FormGroup>
</Form>

<style>
.border-dashed {
  border-style: dashed !important;
}
.bg-light {
  background-color: #f8f9fa !important;
}
.text-truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style> 