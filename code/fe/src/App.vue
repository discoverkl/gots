<template>
  <div id="app" style="width:100%; height: 100%;">
    <!-- <img alt="Code logo" src="./assets/logo.png" /> -->
    <Code class="code" />
  </div>
</template>

<script>
import Code from "./components/Code.vue";

export default {
  name: "App",
  components: {
    Code
  }
};
</script>

<style>
html,
body {
  height: 100%;
  margin: 0;
}
</style>
