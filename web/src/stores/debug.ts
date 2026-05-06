import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useDebugStore = defineStore(
  'debug',
  () => {
    const channelsCollapsed = ref(true);
    const autoRefresh = ref(true);
    const refreshInterval = ref(15);

    function setChannelsCollapsed(val: boolean) {
      channelsCollapsed.value = val;
    }

    function setAutoRefresh(val: boolean) {
      autoRefresh.value = val;
    }

    function setRefreshInterval(val: number) {
      refreshInterval.value = val;
    }

    return {
      channelsCollapsed,
      autoRefresh,
      refreshInterval,
      setChannelsCollapsed,
      setAutoRefresh,
      setRefreshInterval,
    };
  },
  {
    persist: {
      key: 'debug',
      storage: localStorage,
      pick: ['channelsCollapsed', 'autoRefresh', 'refreshInterval'],
    },
  },
);
