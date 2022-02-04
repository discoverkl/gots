import Vue from "vue";
import VueUI from "@vue/ui";
import VCA from "@vue/composition-api";
import App from "./App.vue";
import "@vue/ui/dist/vue-ui.css";

Vue.use(VueUI);

Vue.use(VCA);

Vue.config.productionTip = false;

new Vue({
  render: h => h(App)
}).$mount("#app");
