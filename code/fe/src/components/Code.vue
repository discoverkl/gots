<template>
  <div class="page code-root">
    <div class="tree">
      <ul>
        <TreeItem
          tabindex="100"
          class="root-item"
          ref="treeElement"
          :key="version"
          :item="root"
          :context="state.context"
          :root="true"
          @file:open="handleFileOpen"
          @hook:open="openHook"
          @tree:click="focusTree"
        ></TreeItem>
      </ul>
    </div>
    <div class="file">
      <div class="input" ref="inputElement" @keydown="handleEditorInput" />
      <div class="editor-loading" v-if="loadingMayShow">Loading editor ...</div>
      <div class="editor-mask" :class="{ hide: editingPath }"></div>
    </div>
  </div>
</template>

<script setup lang='ts'>
import { reactive, ref, onMounted } from 'vue'
import api from '../api';
import TreeItem from './TreeItem.vue';

const state = reactive({
  // context is shared by all tree items
  context: {
    selectPath: null, // select path
    emit: () => undefined, // root item's emit
    ready: null, // project ready promise
    notify: null
  }
});
const treeElement = ref(null);
const selectPath = ref(null);
state.context.selectPath = selectPath;

const wrap = f => (title, msg) => f(`${title}: ${msg}`.toLowerCase());

const notify = ref({
  info: wrap(console.info),
  success: wrap(console.info),
  error: wrap(console.error)
});
state.context.notify = notify;

const focusTree = () => {
  treeElement.value.$el.focus();
};

const { openHook } = useOpenHook()

const {
  loadingMayShow,
  inputElement,
  editingPath,
  createEditorOnMounted,
  handleFileOpen,
  focusEditor,
  save: saveFile,
  close: closeFile
} = useFileEditor(notify);

const { openRoot, root, ready, version } = useOpenProject(
  selectPath,
  closeFile
);
state.context.ready = ready;

const { handleEditorInput } = useUserInput(saveFile);

onMounted(function() {
  createEditorOnMounted();
  openRoot();
});

function useFileEditor(notify) {
  const loadingMayShow = ref(false);
  const fileContent = ref('');
  const filePath = ref('');
  const inputElement = ref(null);

  let editor = null;
  let monaco = null;
  let languages = null;

  const createEditorOnMounted = async () => {
    setTimeout(() => {
      if (editor === null) loadingMayShow.value = true;
    }, 1000);

    monaco = await (window as any).getmonaco();
    languages = monaco.languages.getLanguages();
    editor = monaco.editor.create(inputElement.value as any, {
      scrollBeyondLastLine: false,
      automaticLayout: true
    });
    loadingMayShow.value = false;
  };

  const handleFileOpen = async path => {
    filePath.value = '';
    fileContent.value = '';

    try {
      fileContent.value = await api.loadText(path);
      filePath.value = path;
      openFileInEditor();
    } catch (ex) {
      notify.value.error('Open File Failed', `${ex}`);
      return;
    }
  };

  const focusEditor = () => editor.focus();

  const save = async () => {
    if (filePath.value == '') {
      notify.value.error('Save File Failed', 'file path is empty');
      return;
    }

    try {
      await api.saveText(filePath.value, fileContent.value);
      notify.value.success('File Saved', `${filePath.value}`);
    } catch (ex) {
      notify.value.error('Save File Failed', `${ex}`);
      return;
    }
  };

  const close = () => {
    filePath.value = '';
    fileContent.value = '';
    if (editor !== null) editor.setValue('');
  };

  // editor v-model
  const openFileInEditor = () => {
    // update editor
    if (editor !== null) {
      if (filePath.value === '') {
        editor.setValue('');
        return;
      }
      const lang = filename2language(filePath.value);
      editor.setModel(monaco.editor.createModel(fileContent.value, lang));
      editor.getModel()!.onDidChangeContent(e => {
        fileContent.value = editor!.getValue();
      });
      editor.layout();
      // editor.focus();
    }
  };

  const filename2language = (name: string) => {
    name = name.substr(name.lastIndexOf('/') + 1);
    const index = name.lastIndexOf('.');
    if (index === -1) {
      return '';
    }
    const ext = name.substr(index).toLowerCase();
    for (const lang of languages) {
      if (lang.extensions && lang.extensions.indexOf(ext) !== -1) {
        return lang.id;
      }
    }
  };

  return {
    loadingMayShow,
    inputElement,
    editingPath: filePath,
    notify,
    createEditorOnMounted,
    handleFileOpen,
    focusEditor,
    save,
    close
  };
}

function useOpenHook() {
  const openHook = (path: string, ret) => {
    if (!api) return;
    if (path && (path[0] == '/' || path[1] == '\\')) path = path.substr(1);
    switch (path) {
      case '':
      case 'src':
        ret.value = true;
        break;
    }
  };

  return {
    openHook
  };
}

function useOpenProject(selectPath, close) {
  const version = ref(0);
  const root = ref({
    name: '',
    path: '',
    isDir: true
  });

  let readyReslove = null;
  const ready = new Promise((reslove, reject) => {
    readyReslove = reslove;
  });

  const openRoot = () => {
    const name = 'My Files';
    selectPath.value = null;
    close();
    version.value++;
    root.value.name = name;
    root.value = {
      name: name,
      path: '',
      isDir: true
    };
    readyReslove();
  };
  return {
    ready,
    selectPath,
    root,
    version,
    openRoot
  };
}

function useUserInput(save) {
  const handleEditorInput = (e: KeyboardEvent) => {
    if (e.ctrlKey || e.metaKey) {
      switch (e.code) {
        // ctrl/cmd + S
        case 'KeyS':
          save();
          break;
        default:
          return;
      }
      e.preventDefault();
    }
  };

  return {
    handleEditorInput
  };
}
</script>

<style scoped>
.page {
  font-family: 'monaco', 'monospace';
  font-size: 12px;
}
.code-root {
  width: calc(100% - 10px);
  height: calc(100% - 10px);
  padding: 5px;
  display: flex;
  background: white;
}
.root-item {
  cursor: pointer;
  position: relative;
}
.root-item:focus {
  outline: none;
}
ul {
  list-style: none;
  margin: 0;
  padding-left: 0;
}
.tree {
  flex: 0 0 auto;
  min-width: 180px;
  max-width: 500px;
  width: 20%;
  height: 100%;
  overflow: scroll;
  border-right-width: 1px;
  border-right-style: solid;
  border-right-color: lightgray;
}
.file {
  flex: 1 1 auto;
  width: 100%;
  height: 100%;
  position: relative;
}
.input {
  height: 100%;
}
.editor-loading {
  position: absolute;
  top: calc(50% - 30px);
  width: 100%;
  text-align: center;
  font-size: 30px;
  pointer-events: none;
  opacity: 0.382;
}
.editor-mask {
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  background: white;
}
.editor-mask.hide {
  visibility: hidden;
}
</style>
