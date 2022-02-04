<template>
  <div class="page code-root">
    <div class="tree" @keydown="handleTreeInput">
      <ul>
        <tree-item
          tabindex="100"
          class="root-item"
          ref="treeElement"
          :key="version"
          :item="root"
          :context="context"
          :root="true"
          @file:open="handleFileOpen"
          @hook:open="openHook"
          @tree:click="focusTree"
        ></tree-item>
      </ul>
    </div>
    <div class="file">
      <div class="input" ref="inputElement" @keydown="handleEditorInput" />
      <div class="editor-loading" v-if="loadingMayShow">Loading editor ...</div>
      <div class="editor-mask" :class="{ hide: editingPath }"></div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import {
  Ref,
  reactive,
  toRefs,
  ref,
  onMounted,
  watch
} from "@vue/composition-api";
import TreeItem from "./TreeItem.vue";
import api from "../api";

export interface Notify {
  info: (title: string, msg: string) => void;
  success: (title: string, msg: string) => void;
  error: (title: string, msg: string) => void;
}

export interface File {
  name: string;
  path: string;
  children: File[];
  isDir: boolean;
  parent: File;
}

export default {
  components: {
    "tree-item": TreeItem
  },
  setup() {
    const state = reactive({
      // context is shared by all tree items
      context: {
        selectPath: null, // select path
        emit: () => undefined, // root item's emit
        ready: null, // project ready promise
        notify: null
      }
    });
    const treeElement: Ref<any> = ref(null);
    const selectPath: Ref<null | string> = ref(null);
    state.context.selectPath = selectPath;

    const wrap = f => (title, msg) => f(`${title}: ${msg}`.toLowerCase());

    const notify: Ref<Notify> = ref({
      info: wrap(console.info),
      success: wrap(console.info),
      error: wrap(console.error)
    });
    state.context.notify = notify;

    const focusTree = () => {
      treeElement.value.$el.focus();
    };

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

    const { handleEditorInput, handleTreeInput } = useUserInput(
      root,
      saveFile,
      selectPath,
      treeElement,
      handleFileOpen,
      focusEditor
    );

    onMounted(function() {
      createEditorOnMounted();
      openRoot();
    });

    return {
      ...toRefs(state),
      treeElement,
      focusTree,
      root,
      version,
      handleFileOpen,
      inputElement,
      editingPath,
      loadingMayShow,
      handleEditorInput,
      handleTreeInput,
      ...useOpenHook()
    };
  }
};

function useFileEditor(notify: Ref<Notify>) {
  const loadingMayShow = ref(false);
  const fileContent = ref("");
  const filePath = ref("");
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
      scrollBeyondLastLine: false
    });
    loadingMayShow.value = false;
  };

  const handleFileOpen = async path => {
    filePath.value = "";
    fileContent.value = "";

    try {
      fileContent.value = await api.loadText(path);
      filePath.value = path;
      openFileInEditor();
    } catch (ex) {
      notify.value.error("Open File Failed", `${ex}`);
      return;
    }
  };

  const focusEditor = () => editor.focus();

  const save = async () => {
    if (filePath.value == "") {
      notify.value.error("Save File Failed", "file path is empty");
      return;
    }

    try {
      await api.saveText(filePath.value, fileContent.value);
      notify.value.success("File Saved", `${filePath.value}`);
    } catch (ex) {
      notify.value.error("Save File Failed", `${ex}`);
      return;
    }
  };

  const close = () => {
    filePath.value = "";
    fileContent.value = "";
    if (editor !== null) editor.setValue("");
  };

  // editor v-model
  const openFileInEditor = () => {
    // update editor
    if (editor !== null) {
      if (filePath.value === "") {
        editor.setValue("");
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
    name = name.substr(name.lastIndexOf("/") + 1);
    const index = name.lastIndexOf(".");
    if (index === -1) {
      return "";
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
    if (path && (path[0] == "/" || path[1] == "\\")) path = path.substr(1);
    switch (path) {
      case "":
      case "src":
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
    name: "",
    path: "",
    isDir: true
  });

  let readyReslove = null;
  const ready = new Promise((reslove, reject) => {
    readyReslove = reslove;
  });

  const openRoot = () => {
    const name = "My Files";
    selectPath.value = null;
    close();
    version.value++;
    root.value.name = name;
    root.value = {
      name: name,
      path: "",
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

function useUserInput(
  root,
  save,
  selectPath,
  treeElement,
  handleFileOpen,
  focusEditor
) {
  const handleEditorInput = (e: KeyboardEvent) => {
    if (e.ctrlKey || e.metaKey) {
      switch (e.keyCode) {
        // ctrl/cmd + S
        case 83:
          save();
          break;
        default:
          return;
      }
      e.preventDefault();
    }
  };

  const walk = (item, fn): boolean => {
    if (item) {
      if (fn(item) === true) return true;
      if (item.children) {
        for (const child of item.children) {
          if (walk(child, fn) === true) return true;
        }
      }
    }
  };

  const walk2 = (item, fn): boolean => {
    if (item) {
      if (fn(item) === true) return true;
      if (item.$children) {
        for (const child of item.$children) {
          if (walk2(child, fn) === true) return true;
        }
      }
    }
  };

  const path2item = path => {
    let target = null;
    walk(root.value, item => {
      if (item.path == path) {
        target = item;
        return true;
      }
    });
    return target;
  };

  const path2host = path => {
    let target = null;
    walk2(treeElement.value, item => {
      if (item.item.path == path) {
        target = item;
        return true;
      }
    });
    return target;
  };

  const select = (path: string) => {
    selectPath.value = path;
    if (path === null) return;
    let host = path2host(path);
    if (host === null) return;
    host.$el.scrollIntoViewIfNeeded();
  };

  const navigate = (action: string) => {
    const path = selectPath.value;
    if (path === null) return;
    let host = path2host(path);
    if (host === null) return;
    let target = host.item;
    if (target === null) return;

    let hit = false;
    switch (action) {
      case "up": // previous+lastchild... -> previous -> parent
        if (target.parent) {
          const items = target.parent.children;
          const index = items.indexOf(target);
          if (index > 0) {
            target = items[index - 1];
            while (target.isDir) {
              host = path2host(target.path);
              if (host === null) return;
              if (!host.isOpen) break;
              if (!target.children || target.children.length == 0) break;
              target = target.children[target.children.length - 1];
            }
            select(target.path);
            break;
          }
          select(target.parent.path);
          break;
        }
        break;
      case "down": // child -> next -> parent+next
        if (
          target.isDir &&
          host.isOpen &&
          target.children &&
          target.children.length > 0
        ) {
          select(target.children[0].path);
          break;
        }
        while (!hit && target.parent) {
          const items = target.parent.children;
          const index = items.indexOf(target);
          if (index < items.length - 1) {
            select(items[index + 1].path);
            hit = true;
            break;
          }
          target = target.parent;
        }
        break;
      case "left":
        if (target.isDir && host.isOpen) {
          host.toggle();
        } else {
          if (target.parent) select(target.parent.path);
        }
        break;
      case "right":
        if (target.isDir) {
          if (!host.isOpen) host.toggle();
          else {
            if (target.children && target.children.length > 0) {
              select(target.children[0].path);
            }
          }
        }
        break;
      case "space":
        if (target.isDir) host.toggle();
        else {
          handleFileOpen(target.path);
        }
        break;
      case "tab":
        focusEditor();
        break;
    }
  };

  const handleTreeInput = (e: KeyboardEvent) => {
    if (e.ctrlKey || e.metaKey) return;
    switch (e.keyCode) {
      // left
      case 37:
        navigate("left");
        break;
      // up
      case 38:
        navigate("up");
        break;
      // right
      case 39:
        navigate("right");
        break;
      // down
      case 40:
        navigate("down");
        break;
      // space
      case 32:
        navigate("space");
        break;
      // tab
      case 9:
        if (e.shiftKey) return;
        navigate("tab");
        break;
      default:
        return;
    }
    e.preventDefault();
  };
  return {
    handleTreeInput,
    handleEditorInput
  };
}
</script>

<style scoped>
.page {
  font-family: "monaco", "monospace";
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
  min-width: 250px;
  max-width: 500px;
  width: 30%;
  height: 100%;
  overflow: scroll;
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
