<template>
  <TaskList :tasks="tasks">
    <!-- Header -->
    <div slot="header" class="ListHeader">
      <div class="col-md-1">
        <strong>{{ tasksLength }} {{ plural("task", tasksLength)}}</strong>
      </div>
      <div class="col-md-4 flex flex-align-center SearchInput">
        <span class="fa fa-search"></span>
        <input type="text" class="form-control" :value="q" @input="updateQ" @keydown.enter="search">
      </div>
      <div
        class="col-md-1 dropdown"
        :class="{ show: statusFilterOpen }"
        v-click-outside="() => statusFilterOpen = false"
      >
        <button class="btn btn-link dropdown-toggle" @click="statusFilterOpen = !statusFilterOpen">
          Status
        </button>
        <ul class="dropdown-menu">
          <li class="dropdown-item form-check" v-for="status in statuses">
            <label class="form-check-label" :for="`checkbox-status-${status.value}`">
              <input
                type="checkbox"
                :id="`checkbox-status-${status.value}`"
                @change="updateStatusFilter(status.value)"
                :checked="status.checked"
              >
              {{ status.label }}
            </label>
          </li>
        </ul>
      </div>
      <fieldset
        class="col-md-1 dropdown"
        :class="{ show: sortOptionsOpen }"
        v-click-outside="() => sortOptionsOpen = false"
      >
        <button class="btn btn-link dropdown-toggle" @click="sortOptionsOpen = !sortOptionsOpen">
          Sort by
        </button>
        <ul class="dropdown-menu">
          <li class="dropdown-item form-check" v-for="sortOption in sortOptions">
            <label class="form-check-label" :for="`radio-sortOption-${sortOption.value}`">
              <input
                type="radio"
                :id="`radio-sortOption-${sortOption.value}`"
                @change="updateSortOption(sortOption.value)"
                :checked="sortOption.checked"
              >
              {{ sortOption.label }}
            </label>
          </li>
        </ul>
      </fieldset>
      <div class="col-md-1 col-end">
        <i class="fa fa-circle-o-notch fa-spin" v-if="loading"></i>
      </div>
    </div>
  </TaskList>
</template>

<script >
import ClickOutside from 'vue-click-outside';

import { plural } from '@/utils/formats';

import TaskList from '@/components/task-list/TaskList';

import {
  UPDATE_Q,
  UPDATE_STATUS_FILTER,
  UPDATE_SORT_OPTION,
  FETCH_TASKS,
} from './state';

export default {
  name: 'task-list',
  data() {
    return {
      statusFilterOpen: false,
      sortOptionsOpen: false,
    };
  },
  computed: {
    tasks() {
      return this.$store.state.tasks.tasks;
    },
    tasksLength() {
      return this.tasks && this.tasks.length ? this.tasks.length : 0;
    },
    q() {
      return this.$store.state.tasks.q;
    },
    loading() {
      return this.$store.state.tasks.loading;
    },
    statuses() {
      const checkedStatuses = this.$store.state.tasks.statuses;
      return [
        {
          value: 'pending',
          label: 'Pending',
          checked: checkedStatuses.findIndex(s => s === 'pending') !== -1,
        },
        {
          value: 'done',
          label: 'Done',
          checked: checkedStatuses.findIndex(s => s === 'done') !== -1,
        },
        {
          value: "won't do",
          label: "Won't do",
          checked: checkedStatuses.findIndex(s => s === "won't do") !== -1,
        },
      ];
    },
    sortOptions() {
      const sortBy = this.$store.state.tasks.sortBy;
      return [
        {
          value: 'createdAt',
          label: 'Creation date (old to new)',
          checked: !sortBy || sortBy === 'createdAt',
        },
        {
          value: '-createdAt',
          label: 'Creation date (new to old)',
          checked: sortBy === '-createdAt',
        },
        {
          value: 'updatedAt',
          label: 'Last edition date (old to new)',
          checked: sortBy === 'updatedAt',
        },
        {
          value: '-updatedAt',
          label: 'Last edition date (new to old)',
          checked: sortBy === '-updatedAt',
        },
        {
          value: '-priority',
          label: 'Priority (desc)',
          checked: sortBy === '-priority',
        },
        {
          value: '-score',
          label: 'Score (desc)',
          checked: sortBy === '-score',
        },
        {
          value: 'score',
          label: 'Score (asc)',
          checked: sortBy === 'score',
        },
      ];
    },
  },
  methods: {
    plural,
    updateQ(evt) {
      this.$store.commit({ type: UPDATE_Q, q: evt.target.value });
    },
    search() {
      this.$store.dispatch({ type: FETCH_TASKS }).catch();
    },
    updateStatusFilter(status) {
      this.$store.commit({ type: UPDATE_STATUS_FILTER, status });
    },
    updateSortOption(sortBy) {
      this.$store.commit({ type: UPDATE_SORT_OPTION, sortBy });
    },
  },
  components: {
    TaskList,
  },
  directives: {
    ClickOutside,
  },
  watch: {
    tasks() {
      this.$router.push({
        query: {
          q: this.q,
          sortBy: this.$store.state.tasks.sortBy,
          statuses: this.$store.state.tasks.statuses,
        },
      });
    },
  },
};
</script>

<style lang="scss">
@import 'style/_variables';

.SearchInput {
  background: $input-bg;

  border: $input-btn-border-width solid $input-border-color;
  border-radius: $input-border-radius;

  .fa {
    padding-left: $input-padding-x/2;
  }

  input {
    background: transparent;
    border: none;
    padding: $input-padding-y $input-padding-x/2;

    width: 100%;

    &:focus {
      box-shadow: none;
      outline: none;
    }
  }

  &:focus-within {
    border-color: $input-border-focus;
  }
}

.ListHeader {
  width: 100%;

  display: flex;
  align-items: center;

  div:not(:last-child) {
    margin-right: 1rem;
  }

  .col-md-1,
  .col-md-4 {
    padding: 0;
  }

  .btn.btn-link {
    color: $body-color;
  }

  .col-end {
    margin-left: auto;
    text-align: right;
  }

  .form-check {
    margin-bottom: 0;
  }

  .form-check-label {
    padding-left: 0;
  }
}
</style>
