// axios setup with test fallback
const loc = window.location.href.indexOf("http://localhost/evcc/assets/") === 0 ? {
  protocol: "http:",
  hostname: "localhost",
  port: "7070",
} : window.location;

axios.defaults.baseURL = loc.protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + "/api";
axios.defaults.headers.post['Content-Type'] = 'application/json';

//
// State
//

let store = {
  initialized: false,
  state: {
    // configuration
    mode: null,
    soc: null,
    socCapacity: null,
    socTitle: null,
    gridMeter: true,
    pvMeter: true,
    chargeMeter: true,
    phases: null,
    minCurrent: null,
    maxCurrent: null,
    // runtime
    connected: null,
    charging: null,
    gridPower: null,
    pvPower: null,
    chargePower: null,
    chargeDuration: null,
    chargeEstimate: -1,
    chargedEnergy: null,
    socCharge: "—"
  },
  update: function(msg, force) {
    Object.keys(msg).forEach(function(k) {
      if (force || this[k] !== undefined) {
        this[k] = msg[k];
      } else {
        if (k == "error") {
          toasts.error({message: msg[k]});
        } else if (k == "warn") {
          toasts.warn({message: msg[k]});
        } else {
          console.log("invalid key: " + k);
        }
      }
    }, this.state);
  },
  init: function() {
    if (!store.initialized) {
      axios.get("config").then(function(response) {
        store.update(response.data);
        store.initialized = true;
      }).catch(toasts.error);
    }
  }
};

//
// Heartbeat
//

window.setInterval(function() {
  axios.get("health").catch(function(res) {
    res.message = "Server unavailable";
    toasts.error(res)
  });
}, 5000);

//
// Components
//

const toasts = new Vue({
  el: "#toasts",
  data: {
    items: {},
    count: 0,
  },
  methods: {
    raise: function (msg) {
      let found = false;
      Object.keys(this.items).forEach(function (k) {
        let m = this.items[k];
        if (m.type == msg.type && m.message == msg.message) {
          found = true;
        }
      }, this);
      if (!found) {
        msg.id = this.count++;
        Vue.set(this.items, msg.id, msg);
      }
    },
    error: function (msg) {
      msg.type = "error";
      this.raise(msg)
    },
    warn: function (msg) {
      msg.type = "warn";
      this.raise(msg);
    },
    remove: function (msg) {
      Vue.delete(this.items, msg.id);
    },
  }
});

Vue.component('message-toast', {
  props: ['item'],
  template: '#message-template',
  mounted: function () {
    const id = "#message-id-" + this.item.id;
    $(id).toast('show');
    $(id).on('hidden.bs.toast', function () {
      toasts.remove(this.item);
    })
  },
});

Vue.component('modeswitch', {
  template: '#mode-template',
  data: function() {
    return {
      state: store.state // global state
    };
  },
  computed: {
    mode: {
      get: function() {
        return this.state.mode;
      },
      set: function(mode) {
        axios.post('mode/' + mode).then(function(response) {
          this.state.mode = response.data.mode;
        }.bind(this)).catch(error.raise);
      }
    }
  },
});

Vue.component("datapanel", {
  template: "#data-template",
  data: function() {
    return {
      tickerHandle: null,
      state: store.state // global state
    };
  },
  computed: {
    items: function() {
      if (this.state.soc && this.state.pvMeter) {
        return 4;
      } else if (this.state.soc || this.state.pvMeter) {
        return 3;
      } else {
        return 2;
      }
    }
  },
  watch: {
    "state.chargeDuration": function() {
      window.clearInterval(this.tickerHandle);
      if (this.state.charging) {
        // only ticker if actually charging
        this.tickerHandle = window.setInterval(function() {
          this.state.chargeDuration += 1;
        }.bind(this), 1000);
      }
    },
  },
  methods: {
    fmt: function(val) {
      val = Math.abs(val);
      return val >= 100 ? (val / 1e3).toFixed(1) : val.toFixed(0);
    },
    fmtUnit: function(val) {
      return Math.abs(val) >= 100 ? "k" : "";
    },
    fmtDuration: function(d) {
      if (d < 0) {
        return '—';
      }
      var seconds = "0" + (d % 60);
      var minutes = "0" + (Math.floor(d / 60) % 60);
      var hours = "" + Math.floor(d / 3600);
      if (hours.length < 2) {
        hours = "0" + hours;
      }
      return hours + ":" + minutes.substr(-2) + ":" + seconds.substr(-2);
    },
    connect: function() {
      const protocol = loc.protocol == "https:" ? "wss:" : "ws:";
      const uri = protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + "/ws";
      const ws = new WebSocket(uri), self = this;
      ws.onerror = function(evt) {
        ws.close();
      };
      ws.onclose = function(evt) {
        window.setTimeout(self.connect, 1000);
      };
      ws.onmessage = function(evt) {
        try {
          var msg = JSON.parse(evt.data);
          store.update(msg);
        }
        catch (e) {
          console.error(e, evt.data)
        }
      };
    }
  },
  created: function() {
    this.connect();
  },
  destroyed: function() {
    window.clearInterval(this.tickerHandle);
  }
});

//
// Routing
//

const main = Vue.component("main", {
  template: "#main-template",
});

const config = Vue.component("config", {
  template: "#config-template",
  data: function() {
    return {
      state: store.state // global state
    };
  },
});

const embed = Vue.component("embed", {
  template: "#embed-template",
  props: ["title", "subtitle", "img", "iframe", "link"],
});

const routes = [
  { path: "/", component: main },
].concat(routerLinks().map(function(props, idx) {
  return { path: "/links/" + idx, component: embed, props: props }
})).concat([
  { path: "/config", component: config },
]);

const router = new VueRouter({
  routes, // short for `routes: routes`
  linkExactActiveClass: "active" // Bootstrap <nav>
});

const app = new Vue({
  router,
}).$mount("#app");

store.init();
