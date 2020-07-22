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
  data: function () {
    return {
      fmtLimit: 100,
      fmtDigits: 1,
    }
  },
  methods: {
    round: function(num, precision) {
      var base = 10 ** precision;
      return (Math.round(num * base) / base).toFixed(precision);
    },
    fmt: function (val) {
      if (val === undefined || val === null) {
        return 0;
      }
      val = Math.abs(val);
      return val >= this.fmtLimit ? this.round(val / 1e3, this.fmtDigits) : this.round(val, 0);
    },
    fmtUnit: function (val) {
      return Math.abs(val) >= this.fmtLimit ? "k" : "";
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
        tm = hours + ":" + minutes.substr(-2);
      } else {
        var seconds = "0" + (d % 60);
        tm = minutes + ":" + seconds.substr(-2);
      }
      return tm;
    },
    fmtShortDurationUnit: function (d) {
      if (d <= 0 || d == null) {
        return '';
      }
      var hours = Math.floor(d / 3600);
      if (hours >= 1) {
        return "h";
      }
      return "m";
    },
  }
}

//
// State
//

let store = {
  state: {
    availableVersion: null,
    loadpoints: [],
  },
  update: function(msg) {
    let target = this.state;
    if (msg.loadpoint !== undefined) {
      while (this.state.loadpoints.length <= msg.loadpoint) {
        this.state.loadpoints.push({});
      }
      target = this.state.loadpoints[msg.loadpoint];
    }

    Object.keys(msg).forEach(function (k) {
      if (typeof toasts[k] === "function") {
        toasts[k]({message: msg[k]})
      } else {
        Vue.set(target, k, msg[k]);
      }
    });
  },
  init: function() {
    axios.get("config").then(function(msg) {
      for (let i=0; i<msg.data.loadpoints.length; i++) {
        let data = Object.assign(msg.data.loadpoints[i], { loadpoint: i });
        this.update(data);
      }

      delete msg.data.loadpoints;
      this.update(msg.data);
    }.bind(this)).catch(toasts.error);
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
  template: '#message-template',
  props: ['item'],
  mounted: function () {
    const id = "#message-id-" + this.item.id;
    $(id).toast('show');
    $(id).on('hidden.bs.toast', function () {
      toasts.remove(this.item);
    }.bind(this))
  },
});

Vue.component('version', {
  template: '#version-template',
  props: ['installed'],
  data: function () {
    return {
      state: store.state,
    };
  },
  watch: {
    "state.availableVersion": function () {
      if (this.installed != "<<.Version>>" && // go template parsed?
        this.state.availableVersion != this.installed) {
        $(this.$refs.bar).collapse("show");
      }
    }
  }
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
        }.bind(this)).catch(toasts.error);
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
  props: { state: Object },
  mixins: [formatter],
  data: function() {
    return {
      tickerHandle: null,
    };
  },
  watch: {
    "state.chargeDuration": function() {
      window.clearInterval(this.tickerHandle);
      // only ticker if actually charging
      if (this.state.charging && this.state.chargeDuration >= 0) {
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
