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
        <VueIcon
          :class="{ icon: true, show: true, file: !isDir }"
          :icon="iconOfItem"
        />
        <span class="title">{{ item.name }}</span></span
      >
    </div>
    <ul v-show="isOpen" v-if="isDir">
      <tree-item
        class="item"
        v-for="(child, index) in item.children"
        :key="version.toString() + '-' + index"
        :item="child"
        :context="context"
      ></tree-item>
    </ul>
  </li>
</template>

<script lang="ts">
import Vue from "vue";
import {
  ref,
  reactive,
  toRefs,
  computed,
  onMounted
} from "@vue/composition-api";
import api from "../api";
import { Notify, File } from "./Code.vue";

export default {
  name: "tree-item",
  props: {
    item: Object,
    context: Object,
    root: Boolean
  },
  setup(props, { emit }) {
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
              file.parent = props.item;
              items.push(file);
            }
            Vue.set(props.item, "children", items);
          } catch (ex) {
            Vue.set(props.item, "children", null);
            notify.value.error("Open Dir Failed", ex);
          }
        }

        state.isOpen = !state.isOpen;
      }
    }

    const iconOfItem = computed(() => {
      return state.isDir
        ? state.isOpen
          ? "keyboard_arrow_down"
          : "keyboard_arrow_right"
        : "insert_drive_file";
    });

    return {
      ...toRefs(state),
      toggle,
      iconOfItem,
      itemClick
    };
  }
};
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
  width: 25px;
  height: 25px;
  visibility: hidden;
  display: inline-block;
}

.item .icon.show {
  visibility: visible;
}

.item .title {
  line-height: 25px;
  /* font-size: 14px; */
}

.item .icon.file {
  width: 15px;
  height: 25px;
  padding-left: 5px;
  padding-right: 5px;
  /* opacity: 0.8; */
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
