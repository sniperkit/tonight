<template>
  <div id="app">
    <Navbar></Navbar>
    <router-view class="margin-top-1"/>
    <Notifications></Notifications>
  </div>
</template>

<script>
import Navbar from '@/components/nav-bar/NavBar';
import Notifications from '@/modules/notifications/Notifications';

import { LOAD_USER } from '@/modules/user/state';

export default {
  name: 'app',
  components: {
    Navbar,
    Notifications,
  },
  mounted() {
    this.$store.dispatch({ type: LOAD_USER }).catch();
  },
};
</script>
