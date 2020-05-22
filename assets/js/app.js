// axios setup with test fallback
const loc = window.location.href.indexOf("http://localhost/evcc/assets/") === 0 ? {
  protocol: "http:",
  hostname: "localhost",
  port: "7070",
} : window.location;

axios.defaults.baseURL = loc.protocol + "//" + loc.hostname + (loc.port ? ":" + loc.port : "") + "/api";
axios.defaults.headers.post['Content-Type'] = 'application/json';

//
// Mixins
//

let formatter = {
  methods: {
    fmt: function (val) {
      if (val === undefined || val === null) {
        return 0;
      }
      val = Math.abs(val);
      return val >= 100 ? (val / 1e3).toFixed(1) : val.toFixed(0);
    },
    fmtUnit: function (val) {
      return Math.abs(val) >= 100 ? "k" : "";
    },
    fmtDuration: function (d) {
      if (d <= 0 || d == null) {
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
    fmtShortDuration: function (d) {
      if (d <= 0 || d == null) {
        return '—';
      }
      var minutes = (Math.floor(d / 60) % 60);
      var hours = Math.floor(d / 3600);
      var tm;
      if (hours >= 1) {
        minutes = "0" + minutes;
        tm = hours + ":" + minutes.substr(-2) + "h";
      } else {
        var seconds = "0" + (d % 60);
        tm = minutes + ":" + seconds.substr(-2) + "m";
      }
      return tm;
    },
  }
}

//
// State
//

let store = {
  initialized: false,
  state: {
    // configuration
    mode: null,
    gridMeter: null,
    pvMeter: null,
    batteryMeter: null,
    // runtime
    gridPower: null,
    pvPower: null,
    batteryPower: null,
    loadPoints: {
      // configuration
      // soc: null,
      // socCapacity: null,
      // socTitle: null,
      // chargeMeter: true,
      // phases: null,
      // minCurrent: null,
      // maxCurrent: null,
      // runtime
      // connected: null,
      // charging: null,
      // gridPower: null,
      // pvPower: null,
      // chargeCurrent: null,
      // chargePower: null,
      // chargeDuration: null,
      // chargeEstimate: -1,
      // chargedEnergy: null,
      // socCharge: "—"
    },
  },
  update: function(msg) {
    // update loadpoints array
    if (msg.loadpoint !== undefined) {
      for (var i=0; i<this.state.loadPoints.length; i++) {
        if (this.state.loadPoints[i].name != msg.loadpoint) {
          continue;
        }

        Object.keys(msg.data).forEach(function (k) {
          let v = msg.data[k];
          Vue.set(this.state.loadPoints[i], k, v)
        }, this);
      }

      return;
    }

    Object.keys(msg).forEach(function(k) {
      if (k == "error") {
        toasts.error({message: msg[k]});
      } else if (k == "warn") {
        toasts.warn({message: msg[k]});
      } else {
        if (!(k in this.state)) {
          console.log("invalid key: " + k);
        }
        Vue.set(this.state, k, msg[k])
      }
    }, this);
  },
  init: function() {
    if (!store.initialized) {
      axios.get("config").then(function(msg) {
        Object.keys(msg.data).forEach(function (k) {
          let data = {};
          data[k] = msg.data[k];
          store.update(data);
        });

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
        Object.values(this.items, msg.id, msg);
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

Vue.component("site", {
  template: "#site-template",
  mixins: [formatter],
  data: function() {
    return {
      state: store.state // global state
    };
  },
  methods: {
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
  }
});

Vue.component("loadpoint", {
  template: "#loadpoint-template",
  props: ['loadpoint'],
  mixins: [formatter],
  data: function() {
    return {
      tickerHandle: null,
      state: store.state // global state
    };
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
