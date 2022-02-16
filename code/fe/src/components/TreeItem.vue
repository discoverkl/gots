<template>
  <li>
    <div
      :class="{
        item: true,
        active: context.selectPath === item.path
      }"
      @mousedown="itemClick"
    >
      <span class="item-content">
        <!-- <VueIcon
          :class="{ icon: true, show: true, file: !state.isDir }"
          :icon="iconOfItem"
        /> -->
        <span :class="{ icon: true, show: true, file: !state.isDir }" class="material-icons">{{ iconOfItem }}</span>
        <span class="title">{{ item.name }}</span>
      </span>
    </div>
    <ul v-show="state.isOpen" v-if="state.isDir">
      <TreeItem
        class="item"
        v-for="(child, index) in item.children"
        :key="state.version.toString() + '-' + index"
        :item="child"
        :context="context"
      ></TreeItem>
    </ul>
  </li>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted} from "vue";
import api from "../api";
// import { Notify, File } from "./Code.vue";

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

const props = defineProps({
  item: Object,
  context: Object,
  root: Boolean
})

const emit = defineEmits(["hook:open", "tree:click", "file:open"])

// const name = "tree-item"

const state = reactive({
  isOpen: false,
  isDir: computed(() => props.item.isDir),
  version: 0
});

const notify = computed(() => props.context.notify as Notify);

if (props.root === true) {
  props.context.emit = emit;
}

onMounted(async function() {
  // auto open hook
  if (state.isDir) {
    await props.context.ready;
    const ret = ref(false);
    props.context.emit("hook:open", props.item.path, ret);
    if (ret.value === true && !state.isOpen) {
      toggle();
    }
  }
});

async function itemClick(e) {
  if (props.item.isDir) e.preventDefault();
  props.context.emit("tree:click");
  props.context.selectPath = props.item.path;
  await toggle();
  if (!props.item.isDir) {
    props.context.emit("file:open", props.item.path);
  }
}

async function toggle() {
  if (state.isDir) {
    if (!state.isOpen) {
      if (!api) {
        notify.value.error("Open Dir Failed", "API is not ready");
        return;
      }

      try {
        state.version++;
        const files = (await api.listDir(props.item.path)) as File[];
        const items = [];
        for (const file of files) {
          if (file.name.length > 0 && file.name[0] === ".") continue;
          file.parent = props.item as any;
          items.push(file);
        }
        // Vue.set(props.item, "children", items);
        props.item["children"] = items
      } catch (ex) {
        // Vue.set(props.item, "children", null);
        props.item["children"] = null
        notify.value.error("Open Dir Failed", ex);
      }
    }

    state.isOpen = !state.isOpen;
  }
}

const iconOfItem = computed(() => {
  return state.isDir
    ? state.isOpen
      ? "expand_more"
      : "chevron_right"
    : "description";
});

</script>

<style scoped>
ul {
  list-style: none;
  margin-left: 10px;
  padding-left: 0;
}

.item-content {
  white-space: nowrap;
}

.item .icon {
  font-size: 18px;
  visibility: hidden;
  display: inline-block;
  vertical-align: middle;
}

.item .icon.show {
  visibility: visible;
}

.item .title {
  line-height: 25px;
  /* font-size: 14px; */
}

.item .icon.file {
  font-size: 16px;
  padding-left: 4px;
  padding-right: 4px;
}

.item .back {
  background: yellow;
  width: 100%;
  height: 100%;
  z-index: 10;
}

div.item:hover {
  background: #bbe6d630;
}

div.item.active {
  background: #bbe6b6;
}
</style>
