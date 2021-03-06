import { NOTIFICATION_SUCCESS, NOTIFICATION_FAILURE } from './events';

const ADD_NOTIFICATION = 'ADD_NOTIFICATION';
const REMOVE_NOTIFICATION = 'REMOVE_NOTIFICATION';

let notificationID = 0;

const pushNotification = (context, { text }, kind) => {
  notificationID += 1;
  const id = notificationID;

  context.commit({
    type: ADD_NOTIFICATION,
    id,
    kind,
    text,
  });

  setTimeout(() => {
    context.commit({ type: REMOVE_NOTIFICATION, id });
  }, context.state.hideAfter);
};

const getters = {};

const mutations = {
  [ADD_NOTIFICATION]: (state, { id, kind, text }) => {
    state.notifications.push({ id, kind, text });
    return id;
  },
  [REMOVE_NOTIFICATION]: (state, { id }) => {
    const idx = state.notifications.findIndex(n => n.id === id);
    if (idx !== -1) {
      state.notifications.splice(idx, 1);
    }
  },
};

const actions = {
  [NOTIFICATION_SUCCESS]: (context, data) =>
    pushNotification(context, data, 'success'),
  [NOTIFICATION_FAILURE]: (context, data) =>
    pushNotification(context, data, 'danger'),
};

export default {
  state: {
    hideAfter: 5000,
    notifications: [],
  },
  getters,
  mutations,
  actions,
};
