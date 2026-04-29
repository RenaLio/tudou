import './assets/theme.css';
import './styles/tailwind.css';
import './styles/main.scss';

import { VueQueryPlugin, } from '@tanstack/vue-query';
import { MotionPlugin, } from '@vueuse/motion';
import { createPinia, } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import { createApp, } from 'vue';

import App from './App.vue';
import router from './router';

const app = createApp(App,);

const pinia = createPinia();
pinia.use(piniaPluginPersistedstate,);

app.use(pinia,);
app.use(router,);
app.use(VueQueryPlugin,);
app.use(MotionPlugin,);

app.mount('#app',);
