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
        this.installed != "0.0.1-alpha" && // make used?
        this.state.availableVersion != this.installed) {
        $(this.$refs.bar).collapse("show");
      }
    }
  }
});

Vue.component('site', {
  template: '#site-template',
  props: ['state'],
  mixins: [formatter],
  computed: {
    multi: function() {
      return this.state.loadpoints.length > 1
    }
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
          toasts.error(e, evt.data)
        }
      };
    }
  },
  created: function() {
    this.connect();
  }
});

Vue.component("site-details", {
  template: "#site-details-template",
  props: ["state"],
  mixins: [formatter]
});

Vue.component("loadpoint", {
  template: "#loadpoint-template",
  props: ["state", "id", "pv", "multi"],
  mixins: [formatter],
  data: function() {
    return {
      tickerHandle: null,
    };
  },
  computed: {
    hasTargetSoC: function () {
      return this.state.socLevels != null && this.state.socLevels.length > 0;
    },
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
  methods: {
    api: function (func) {
      return "loadpoints/" + this.id + "/" + func;
    },
    targetMode: function (mode) {
      axios.post(this.api("mode") + "/" + mode).then(function (response) {
        this.state.mode = response.data.mode;
      }.bind(this)).catch(toasts.error);
    },
    targetSoC: function (soc) {
      axios.post(this.api("targetsoc") + "/" + soc).then(function (response) {
        this.state.targetSoC = response.data.targetSoC;
      }.bind(this)).catch(toasts.error);
    },
  },
  destroyed: function() {
    window.clearInterval(this.tickerHandle);
  }
});

Vue.component("loadpoint-details", {
  template: "#loadpoint-details-template",
  props: ["state"],
  mixins: [formatter]
});

Vue.component("vehicle", {
  template: "#vehicle-template",
  props: ["state"]
});

Vue.component("mode", {
  template: "#mode-template",
  props: ["mode", "pv", "caption"],
  methods: {
    targetMode: function (mode) {
      this.$emit("updated", mode)
    }
  },
});

Vue.component("soc", {
  template: "#soc-template",
  props: ["soc", "caption", "levels"],
  computed: {
    levelsOrDefault: function() {
      if (this.levels == null || this.levels.length == 0) {
        return []; // disabled, or use 30, 50, 80, 100
      }
      return this.levels;
    }
  },
  methods: {
    targetSoC: function (mode) {
      this.$emit("updated", mode)
    }
  },
});

//
// Routing
//

const main = Vue.component('main', {
  template: '#main-template',
  data: function() {
    return {
      state: store.state  // global state
    }
  },
  methods: {
    configured: function(val) {
      if (val == '<<.Configured>>' && val != '0') {
        return 0+val;
      }
      return false;
    }
  }
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
